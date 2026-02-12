// ==============================================================================
// RaidX Kabaddi Scorer - Frontend (UI Only)
// ==============================================================================
// 
// ARCHITECTURE:
// - All Kabaddi scoring calculations, player status management, revivals, 
//   all-out detection, do-or-die logic, super tackles, and bonus points are
//   handled on the BACKEND (Go server in internal/handlers/matches.go)
// 
// - This frontend is responsible ONLY for:
//   1. Displaying the UI (teams, scores, players, raid info)
//   2. Capturing user actions (selecting raiders/defenders, raid outcomes)
//   3. Sending action payloads to the backend via WebSocket
//   4. Receiving complete game state updates from backend
//   5. Rendering the updated state
//
// - The backend is the single source of truth for all game state
// ==============================================================================

// Constants
const MATCH_STORAGE_KEY = 'currentMatchId';

// State variables
let socket = null;
let matchId = null;
let jwtToken = getValidToken(); // Use getValidToken from auth.js
let teamA = { name: "", score: 0, players: [] };
let teamB = { name: "", score: 0, players: [] };
let playerStats = {};
let game = true;
let selectedRaider = null;
let selectedDefenders = [];
let bonusTaken = false;
let emptyRaidCountA = 0;
let emptyRaidCountB = 0;
let isDoOrDieRaid = false;
let currentRaidNumber = 1;

/**
 * WebSocket Connection Management
 */
// Only allow /scorer access if authenticated
if (!isAuthenticated()) {
    const currentUrl = encodeURIComponent(window.location.href);
    window.location.href = `/login?returnUrl=${currentUrl}`;
}

function setupWebSocket() {
    if (socket !== null) {
        console.log("WebSocket already exists");
        return;
    }

    // Use current hostname for WebSocket connection
    const wsProto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProto}//${location.host}/ws/scorer?token=${jwtToken}`;
    socket = new WebSocket(wsUrl);
    console.log("Setting up WebSocket connection with auth token...");

    socket.onerror = (error) => {
        if (!jwtToken) {
            console.error("No JWT token found, redirecting to login...");
            window.location.href = "/login";
            return;
        }
        console.error("WebSocket error:", error);
        updateConnectionStatus('error');
    };

    // The 'onopen' handler is set later in DOMContentLoaded after teams are loaded.

    socket.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            // Server requests client to initialize game state on first connection
            if (msg.type === 'requestInit') {
                const initialState = {
                    type: 'initialState',
                    data: {
                        teamA: { name: teamA.name, score: teamA.score },
                        teamB: { name: teamB.name, score: teamB.score },
                        playerStats: playerStats,
                        teamAPlayerIds: teamA.players.map(p => p.id),
                        teamBPlayerIds: teamB.players.map(p => p.id),
                        raidNumber: currentRaidNumber,
                        emptyRaidCounts: { teamA: emptyRaidCountA, teamB: emptyRaidCountB }
                    }
                };
                socket.send(JSON.stringify(initialState));
                return;
            }
            if (msg.error) {
                alert(`Server error: ${msg.error}`);
                return;
            }
            if (msg.data) {
                // Backend sends complete calculated state - just update UI
                if (msg.data.teamA) teamA.score = msg.data.teamA.score;
                if (msg.data.teamB) teamB.score = msg.data.teamB.score;
                if (msg.data.playerStats) {
                    playerStats = msg.data.playerStats;
                    // sync the per-player `status` into the team player objects
                    syncPlayerStatusesFromPlayerStats();
                }
                if (msg.data.raidNumber) currentRaidNumber = msg.data.raidNumber;

                // Update empty raid counts from backend (server is source of truth)
                if (msg.data.emptyRaidCounts) {
                    emptyRaidCountA = msg.data.emptyRaidCounts.teamA;
                    emptyRaidCountB = msg.data.emptyRaidCounts.teamB;
                }

                updateDisplay();
                updateRaidInfoUI();
                nextRaid(); // Reset UI selections for the next raid
            }
        } catch (e) {
            console.error('Invalid WS message', e);
        }
    };
    socket.onerror = (error) => {
        console.error("WebSocket error:", error);
        setConnectionStatus('Error');
    };
    
    socket.onclose = () => {
        console.log("WebSocket connection closed.");
        setConnectionStatus('Disconnected');
    };
}

// UI helpers for match status
function setConnectionStatus(status) {
    try {
        const el = document.getElementById('match-status-conn');
        if (!el) return;
        el.textContent = status;
        el.style.background = status === 'Connected' ? '#10b981' : status === 'Error' ? '#ef4444' : '#6b7280';
    } catch (e) { /* ignore */ }
}

function setMatchIdDisplay(id) {
    try {
        const el = document.getElementById('match-status-id');
        if (!el) return;
        el.textContent = `Match: ${id || '—'}`;
    } catch (e) { /* ignore */ }
}

/**
 * Game State Management (UI helpers only - all calculations done on backend)
 */
function initializePlayerStats(team) {
    team.players.forEach(player => {
        playerStats[player.id] = {
            name: player.name,
            id: player.id,
            totalPoints: 0,
            raidPoints: 0,
            defencePoints: 0,
            status: player.status,
        };
    });
}

function getDefendingTeam() {
    return currentRaidNumber % 2 !== 0 ? teamB : teamA;
}

function getRaidingTeam() {
    return currentRaidNumber % 2 !== 0 ? teamA : teamB;
}


function handleLobbyTouch(player, isRaiderTouchingLobby) {
    if (!player) return alert("Select a player first.");
    
    const raidingTeam = getRaidingTeam();
    const defendingTeam = getDefendingTeam();

    // Determine the team that gets the point
    const scoringTeam = isRaiderTouchingLobby ? defendingTeam : raidingTeam;
    
    // Create the payload for the server (backend will handle all calculations)
    const lobbyPayload = {
        type: "lobbyTouch",
        data: {
            touchedPlayerId: player.id,
            isRaider: raidingTeam.players.some(p => p.id === player.id),
            scoringTeam: scoringTeam.name === teamA.name ? "A" : "B"
        }
    };

    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(lobbyPayload));
    } else {
        alert('Socket not connected');
    }
}


function endGame() {
    game = false;
    let message = "";

    // Check if teams are properly initialized
    if (!teamA || !teamB || typeof teamA.score === 'undefined' || typeof teamB.score === 'undefined') {
        console.error('Teams not properly initialized');
        return;
    }

    if (teamA.score > teamB.score) {
        message = `${teamA.name || 'Team A'} wins`;
    } else if (teamA.score < teamB.score) {
        message = `${teamB.name || 'Team B'} wins`;
    } else {
        message = "It was a tie";
    }

    // Update live commentary with end match message
    const commentaryEl = document.getElementById('live-commentary');
    if (commentaryEl) {
        commentaryEl.textContent = `🏁 Match Ended: ${message}`;
    }

    alert(message);

    // Get token for authentication
    const token = getValidToken();
    if (!token) {
        console.error('No authentication token found');
        window.location.href = '/login';
        return;
    }

    // Derive playerId from token payload (use helper) and fall back to URL path
    let playerId = (typeof getUserIdFromToken === 'function') ? getUserIdFromToken() : null;
    if (!playerId) {
        // fallback: try to extract from URL path (legacy flows)
        playerId = window.location.pathname.split("/").pop();
    }

    // Clear stored match id so refresh won't attempt to rejoin a finished match
    try { localStorage.removeItem(MATCH_STORAGE_KEY); } catch (e) { /* ignore */ }

    // Send API request to end game with authentication
    const eventId = new URLSearchParams(window.location.search).get('event_id');
    const tournamentId = new URLSearchParams(window.location.search).get('tournament_id');
    const fixtureId = new URLSearchParams(window.location.search).get('fixture_id');
    const eventParam = eventId ? `&event_id=${encodeURIComponent(eventId)}` : '';
    const tournamentParam = tournamentId ? `&tournament_id=${encodeURIComponent(tournamentId)}` : '';
    const fixtureParam = fixtureId ? `&fixture_id=${encodeURIComponent(fixtureId)}` : '';
    fetch(`/api/endgame?match_id=${encodeURIComponent(matchId)}${eventParam}${tournamentParam}${fixtureParam}`, {
        method: 'GET',
        headers: {
            'Authorization': `Bearer ${token}`
        }
    }).then(response => {
        if (response.ok) {
            // Successfully ended game, redirect appropriately
            if (tournamentId) {
                // Redirect to tournament dashboard
                window.location.href = `/organizer/tournament?id=${tournamentId}&token=${encodeURIComponent(token)}`;
            } else if (eventId) {
                window.location.href = `/organizer/event/${eventId}?token=${encodeURIComponent(token)}`;
            } else {
                window.location.href = `/organizer/events?token=${encodeURIComponent(token)}`;
            }
        } else {
            throw new Error('Failed to end game');
        }
    }).catch(error => {
        console.error('Error ending game:', error);
        alert('Failed to end game. Please try again.');
    });
}


function nextRaid() {
    selectedRaider = null;
    selectedDefenders = [];
    bonusTaken = false;
    isDoOrDieRaid = false;

    // Reset UI elements
    document.getElementById("bonus-toggle").checked = false;
    document.getElementById("bonus-toggle").disabled = true;

    updateDisplay();
    updateCurrentRaidDisplay();
    updateBonusToggleVisibility();
    updateRaidInfoUI();
}


/**
 * UI Interaction Handlers
 */
function handlePlayerClick(playerId) {
    const currentTeam = getRaidingTeam();
    const opposingTeam = getDefendingTeam();

    let player = [...currentTeam.players, ...opposingTeam.players].find(p => p.id === playerId);

    if (!player || player.status !== "in") return;

    if (currentTeam.players.some(p => p.id === player.id)) {
        // Only allow selection of the raider from the raiding team
        selectedRaider = player;
        selectedDefenders = []; // Reset defenders when a new raider is selected
    } else {
        // Toggle defender selection from the defending team
        if (selectedRaider) {
            toggleDefenderSelection(player);
        } else {
            alert("Please select a raider first.");
        }
    }

    updateCurrentRaidDisplay();
    updateBonusToggleVisibility();
}

function toggleDefenderSelection(player) {
    if (selectedDefenders.some(p => p.id === player.id)) {
        selectedDefenders = selectedDefenders.filter(p => p.id !== player.id);
    } else {
        selectedDefenders.push(player);
    }
}

/**
 * Score Action Handlers (Send RAW data only - backend determines everything)
 */
function raidSuccessful() {
    if (!selectedRaider) {
        alert("Select a raider.");
        return;
    }

    // Send ONLY raw player IDs and action - backend calculates everything
    const payload = {
        raidType: "successful",
        raiderId: selectedRaider.id,
        defenderIds: selectedDefenders.map(d => d.id),
        bonusTaken: bonusTaken
    };

    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(payload));
    } else {
        alert('Socket not connected');
    }
}

function defenseSuccessful() {
    if (!selectedRaider) {
        alert("Select a raider.");
        return;
    }

    // Send ONLY raw player IDs and action - backend calculates everything
    const payload = {
        raidType: "defense",
        raiderId: selectedRaider.id,
        defenderIds: selectedDefenders.map(d => d.id),
        bonusTaken: bonusTaken
    };
    
    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(payload));
    }
}

function emptyRaid() {
    if (!selectedRaider) {
        alert("Select a raider.");
        return;
    }

    // Send ONLY raw player ID and action - backend tracks empty raid counts
    const payload = {
        raidType: "empty",
        raiderId: selectedRaider.id,
        defenderIds: [],
        bonusTaken: bonusTaken
    };
    
    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(payload));
    }
}


/**
 * UI Rendering and Updates
 */
function updateCurrentRaidDisplay() {
    let display = document.getElementById("current-raid");
    const raidingTeam = getRaidingTeam();
    const defendingTeam = getDefendingTeam();
    
    if (!selectedRaider) {
        display.innerHTML = `**${raidingTeam.name}** to raid. Select a raider.`;
    } else {
        const defendersList = selectedDefenders.map(p => p.name).join(", ");
        display.innerHTML = `Raider (**${selectedRaider.name}** from ${raidingTeam.name}), Defended By: ${defendersList || 'No defenders selected'}`;
    }
}

function updateDisplay() {
    document.getElementById("teamA-score").textContent = teamA.score;
    document.getElementById("teamB-score").textContent = teamB.score;
    renderPlayers();
}

function renderPlayers() {
    const render = (team, containerId) => {
        const container = document.getElementById(containerId);
        container.innerHTML = "";
        team.players.forEach(player => {
            const btn = document.createElement("button");

            btn.className = `player-card btn`;

            if (player.status === "in") {
                btn.classList.add("btn-outline-primary");
            } else {
                btn.classList.add("btn-secondary");
                btn.style.textDecoration = "line-through";
                btn.disabled = true;
                btn.style.opacity = "0.6";
            }

            // Highlight selected raider/defenders
            if (selectedRaider && selectedRaider.id === player.id) {
                 btn.classList.add("btn-success");
            } else if (selectedDefenders.some(p => p.id === player.id)) {
                 btn.classList.add("btn-warning");
            }


            btn.textContent = player.name;
            btn.onclick = () => handlePlayerClick(player.id);

            container.appendChild(btn);
        });
    };

    render(teamA, "teamA-players");
    render(teamB, "teamB-players");
}

// Sync statuses from the authoritative playerStats map (received from backend server)
// into the team player objects used for rendering and selection.
function syncPlayerStatusesFromPlayerStats() {
    try {
        [teamA, teamB].forEach(team => {
            team.players.forEach(p => {
                if (playerStats && playerStats[p.id] && playerStats[p.id].status) {
                    p.status = playerStats[p.id].status;
                }
            });
        });
        // If a currently selected raider was marked out by backend, clear selection
        if (selectedRaider && playerStats[selectedRaider.id] && playerStats[selectedRaider.id].status !== 'in') {
            selectedRaider = null;
        }
        // Remove any selected defenders who are now out (as determined by backend)
        selectedDefenders = selectedDefenders.filter(d => playerStats[d.id] && playerStats[d.id].status === 'in');
        // Refresh UI to reflect possible deselections
        updateCurrentRaidDisplay();
        renderPlayers();
    } catch (e) {
        console.error('Error syncing player statuses:', e);
    }
}

function updateBonusToggleVisibility() {
    const bonusToggle = document.getElementById("bonus-toggle");
    const defendingTeam = getDefendingTeam();
    // Bonus is only available if there are 6 or more 'in' players in the defending team
    const inPlayers = defendingTeam.players.filter(p => p.status === "in").length;

    bonusToggle.disabled = inPlayers < 6;
}

function updateRaidInfoUI() {
    const raidElement = document.getElementById("raid-number");
    if (!raidElement) {
        console.warn("⚠️ 'raid-number' element not found");
        return;
    }

    const raidingTeam = getRaidingTeam();
    const emptyCount = raidingTeam.name === teamA.name ? emptyRaidCountA : emptyRaidCountB;
    const raidType = emptyCount === 2 ? "🔴 Do or Die Raid" : "Normal Raid";

    raidElement.innerHTML = `Raid: **${currentRaidNumber}** | Turn: **${raidingTeam.name}** | Status: **${raidType}**`;
}

/**
 * Initialization (Runs when the HTML document is fully loaded)
 */
document.addEventListener("DOMContentLoaded", async () => {
    // Robust clipboard helper (navigator.clipboard with textarea fallback)
    function copyToClipboard(text) {
        if (!text) return Promise.reject(new Error('No text to copy'));
        if (navigator.clipboard && navigator.clipboard.writeText) {
            return navigator.clipboard.writeText(text);
        }
        return new Promise((resolve, reject) => {
            try {
                const ta = document.createElement('textarea');
                ta.value = text;
                // Avoid scrolling to bottom
                ta.style.position = 'fixed';
                ta.style.left = '-9999px';
                document.body.appendChild(ta);
                ta.focus();
                ta.select();
                const ok = document.execCommand('copy');
                document.body.removeChild(ta);
                if (ok) resolve(); else reject(new Error('execCommand failed'));
            } catch (err) {
                reject(err);
            }
        });
    }

    // Wire Copy Match Link button (top persistent button)
    const copyMatchIdBtn = document.getElementById('copy-matchid-btn');
    if (copyMatchIdBtn) {
        copyMatchIdBtn.addEventListener('click', () => {
            const id = matchId || '';
            if (!id) return;
            const matchLink = `${location.origin}/viewer?match_id=${encodeURIComponent(id)}`;
            copyToClipboard(matchLink).then(() => {
                copyMatchIdBtn.textContent = 'Copied!';
                setTimeout(() => (copyMatchIdBtn.textContent = 'Copy Match Link'), 1200);
            }).catch(() => alert('Copy failed - please copy manually'));
        });
    }
    const params = new URLSearchParams(window.location.search);
    const normalizeParam = (value) => {
        if (!value) return null;
        if (value === 'null' || value === 'undefined') return null;
        return value;
    };
    const team1Id = normalizeParam(params.get("team1_id"));
    const team2Id = normalizeParam(params.get("team2_id"));
    // Allow prefilled match id via URL (scorer could open /scorer?match_id=xxx)
    const prefillMatchId = params.get("match_id");
    // Optional resume flag to auto-join without showing overlay
    const resumeFlag = params.get("resume") === "1";
    let autoJoinedFromPrefill = false;

    console.log("DOM fully loaded. Starting initialization.");

    if (team1Id && team2Id) {
        try {
            // 1. Fetch Team Data
            const authToken = getValidToken && getValidToken();
            const authHeaders = authToken ? { 'Authorization': `Bearer ${authToken}` } : {};
            const [res1, res2] = await Promise.all([
                fetch(`/api/team/${team1Id}`, { headers: authHeaders }),
                fetch(`/api/team/${team2Id}`, { headers: authHeaders })
            ]);

            const data1 = await res1.json();
            const data2 = await res2.json();

            // Load selected players from localStorage
            const selectedA = JSON.parse(localStorage.getItem("teamA_selected"));
            const selectedB = JSON.parse(localStorage.getItem("teamB_selected"));

            // 2. Initialize Teams and Players
            const team1Name = normalizeParam(params.get("team1_name"));
            const team2Name = normalizeParam(params.get("team2_name"));

            const normalizePlayer = (p) => {
                const id = p._id || p.id || p.userId || p.user_id;
                const name = p.name || p.fullName || p.full_name || p.email || 'Unnamed Player';
                return { id, name, status: "in" };
            };

            teamA.name = team1Name || data1.team_name || data1.TeamName || "Team A";
            // Ensure players array contains the required 'status' field
            teamA.players = selectedA ? selectedA.map(normalizePlayer) : [];
            teamB.name = team2Name || data2.team_name || data2.TeamName || "Team B";
            teamB.players = selectedB ? selectedB.map(normalizePlayer) : [];
            
            // Update the UI with team names immediately
            document.getElementById("teamA-name").textContent = teamA.name;
            document.getElementById("teamB-name").textContent = teamB.name;
            document.getElementById("teamA-header").textContent = teamA.name;
            document.getElementById("teamB-header").textContent = teamB.name;
            
            // Log team initialization for debugging
            console.log('Teams initialized:', { 
                teamA: { name: teamA.name, playerCount: teamA.players.length },
                teamB: { name: teamB.name, playerCount: teamB.players.length }
            });

            // 3. Initialize Player Stats
            initializePlayerStats(teamA);
            initializePlayerStats(teamB);

            // 4. If a match id was provided via URL, prefill it into the match setup input
            if (prefillMatchId) {
                const el = document.getElementById('match-id-input');
                if (el) el.value = prefillMatchId;
            } else {
                // generate a friendly short id and prefill the input
                const shortId = 'm-' + Date.now().toString(36) + Math.random().toString(36).slice(2,6);
                const el = document.getElementById('match-id-input');
                if (el) el.value = shortId;
            }

            // Wire UI for match setup actions
            const copyBtn = document.getElementById('copy-match-link');
            const startBtn = document.getElementById('start-match-btn');
            const matchInput = document.getElementById('match-id-input');
            const viewerLinkEl = document.getElementById('viewer-link');

            function updateViewerLink() {
                if (!matchInput || !viewerLinkEl) return;
                // Prefer the server route /viewer so route-level rendering is used
                const link = `${location.origin}/viewer?match_id=${encodeURIComponent(matchInput.value)}`;
                viewerLinkEl.textContent = link;
            }

            if (matchInput) {
                matchInput.addEventListener('input', updateViewerLink);
                updateViewerLink();
            }

            if (copyBtn) {
                copyBtn.addEventListener('click', () => {
                    updateViewerLink();
                    const link = viewerLinkEl ? viewerLinkEl.textContent : '';
                    if (!link) return;
                    copyToClipboard(link).then(() => {
                        copyBtn.textContent = 'Copied';
                        setTimeout(() => (copyBtn.textContent = 'Copy Link'), 1500);
                    }).catch(() => alert('Copy failed - please copy manually'));
                });
            }

            if (startBtn) {
                startBtn.addEventListener('click', () => {
                    if (!matchInput || !matchInput.value) return alert('Please enter a match ID');
                    matchId = matchInput.value.trim();
                    // persist match id so refreshes keep the same match
                    try { localStorage.setItem(MATCH_STORAGE_KEY, matchId); } catch (e) { console.warn('Failed to persist match id', e); }
                    // update UI
                    setMatchIdDisplay(matchId);
                    // hide setup overlay
                    const overlay = document.getElementById('match-setup');
                    if (overlay) overlay.style.display = 'none';
                    // initialize websocket and join
                    setupWebSocket();
                    if (socket) {
                        socket.onopen = () => {
                            socket.send(JSON.stringify({ type: 'join', matchId }));
                            setConnectionStatus('Connected');
                            console.log('Joined match:', matchId);
                        };
                    }
                });
            }

            // If match_id is provided in URL, auto-join using that id (tournament flow)
            if (prefillMatchId) {
                matchId = prefillMatchId.trim();
                if (matchInput) matchInput.value = matchId;
                try { localStorage.setItem(MATCH_STORAGE_KEY, matchId); } catch (e) { console.warn('Failed to persist match id', e); }
                setMatchIdDisplay(matchId);
                const overlay = document.getElementById('match-setup');
                if (overlay) overlay.style.display = 'none';
                setupWebSocket();
                if (socket) {
                    socket.onopen = () => {
                        socket.send(JSON.stringify({ type: 'join', matchId }));
                        setConnectionStatus('Connected');
                        console.log('Auto-joined match:', matchId);
                    };
                }
                autoJoinedFromPrefill = true;
            }

            // If a match id was previously stored, auto-join that match (persistence across refreshes)
            const stored = (() => { try { return localStorage.getItem(MATCH_STORAGE_KEY); } catch (e) { return null; } })();
            if (!autoJoinedFromPrefill && stored && stored.trim() !== '') {
                matchId = stored.trim();
                if (matchInput) matchInput.value = matchId;
                // Only auto-join (and hide overlay) if explicitly allowed via URL (prefill/resume)
                const allowAutoJoin = !!prefillMatchId || resumeFlag;
                if (allowAutoJoin) {
                    const overlay = document.getElementById('match-setup');
                    if (overlay) overlay.style.display = 'none';
                    setMatchIdDisplay(matchId);
                    setupWebSocket();
                    if (socket) {
                        socket.onopen = () => {
                            socket.send(JSON.stringify({ type: 'join', matchId }));
                            setConnectionStatus('Connected');
                            console.log('Auto-joined stored match:', matchId);
                        };
                    }
                } else {
                    // Prefill but keep overlay so user can copy/confirm
                    setMatchIdDisplay('—');
                    const overlay = document.getElementById('match-setup');
                    if (overlay) overlay.style.display = 'flex';
                }
            }

        } catch (err) {
            console.error("Failed to load teams:", err);
            // Alert user if data loading fails
            alert("Error: Failed to load team data. Check console for details.");
        }
    } else {
        console.error('Missing team IDs in scorer URL. team1_id/team2_id are required.');
    }

    // 6. Setup Static UI Elements and Listeners
    document.getElementById("teamA-name").textContent = teamA.name;
    document.getElementById("teamB-name").textContent = teamB.name;
    document.getElementById("teamA-header").textContent = teamA.name;
    document.getElementById("teamB-header").textContent = teamB.name;

    document.getElementById("teamA-score").textContent = teamA.score;
    document.getElementById("teamB-score").textContent = teamB.score;

    document.getElementById("bonus-toggle").addEventListener("change", () => {
        bonusTaken = document.getElementById("bonus-toggle").checked;
    });

    document.getElementById("raider-lobby-entry").addEventListener("click", function () {
        handleLobbyTouch(selectedRaider, true);
    });

    document.getElementById("defender-lobby-entry").addEventListener("click", function () {
        // Assume first selected defender is the one touching the lobby
        if (selectedDefenders.length > 0) {
            handleLobbyTouch(selectedDefenders[0], false);
        } else {
            alert("Select at least one defender.");
        }
    });

    // Initial UI render
    renderPlayers();
    updateBonusToggleVisibility();
    updateRaidInfoUI();
    updateCurrentRaidDisplay(); // Initial raid info

});