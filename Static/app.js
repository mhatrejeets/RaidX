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
let tossWinner = null; // 'teamA' | 'teamB'
let tossDecision = 'raid'; // 'raid' | 'defend'
let firstRaidingTeam = 'teamA';
let requireServerRosterHydration = false;
let serverRosterHydrated = false;

function hydrateTeamsFromServerState(serverData) {
    if (!serverData || !serverData.playerStats) return false;

    const statsMap = serverData.playerStats || {};
    let teamAIds = Array.isArray(serverData.teamAPlayerIds) ? serverData.teamAPlayerIds.filter(Boolean) : [];
    let teamBIds = Array.isArray(serverData.teamBPlayerIds) ? serverData.teamBPlayerIds.filter(Boolean) : [];

    if (teamAIds.length === 0 && teamBIds.length === 0) {
        const allIds = Object.keys(statsMap);
        teamAIds = allIds.slice(0, 7);
        teamBIds = allIds.slice(7, 14);
    }

    const buildPlayer = (playerId) => {
        const stat = statsMap[playerId] || {};
        return {
            id: playerId,
            name: stat.name || stat.Name || playerId,
            status: stat.status || stat.Status || 'in'
        };
    };

    if (teamAIds.length > 0) {
        teamA.players = teamAIds.map(buildPlayer);
    }
    if (teamBIds.length > 0) {
        teamB.players = teamBIds.map(buildPlayer);
    }

    const teamAName = (serverData.teamA && serverData.teamA.name) || teamA.name || 'Team A';
    const teamBName = (serverData.teamB && serverData.teamB.name) || teamB.name || 'Team B';

    teamA.name = teamAName;
    teamB.name = teamBName;

    const teamANameEl = document.getElementById('teamA-name');
    const teamBNameEl = document.getElementById('teamB-name');
    const teamAHeaderEl = document.getElementById('teamA-header');
    const teamBHeaderEl = document.getElementById('teamB-header');
    if (teamANameEl) teamANameEl.textContent = teamAName;
    if (teamBNameEl) teamBNameEl.textContent = teamBName;
    if (teamAHeaderEl) teamAHeaderEl.textContent = teamAName;
    if (teamBHeaderEl) teamBHeaderEl.textContent = teamBName;

    serverRosterHydrated = true;
    return true;
}

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
            if (msg.type === 'scorerTakeover') {
                const takeoverMsg = msg.message || 'Continued on other device';
                const redirectUrl = msg.redirectUrl || '/organizer/dashboard';
                alert(takeoverMsg);
                window.location.href = redirectUrl;
                return;
            }
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
                        emptyRaidCounts: { teamA: emptyRaidCountA, teamB: emptyRaidCountB },
                        tossWinner: tossWinner,
                        tossDecision: tossDecision,
                        firstRaidingTeam: firstRaidingTeam
                    }
                };
                socket.send(JSON.stringify(initialState));
                return;
            }
            if (msg.error) {
                const errText = String(msg.error || '');
                if (errText.toLowerCase().includes('already being scored')) {
                    setConnectionStatus('Locked');
                    showScorerLockNotice('This match is locked by other device.');
                    try { socket.close(); } catch (e) { /* ignore */ }
                    return;
                }
                alert(`Server error: ${msg.error}`);
                return;
            }
            if (msg.data) {
                // Backend sends complete calculated state - just update UI
                if (msg.data.teamA) teamA.score = msg.data.teamA.score;
                if (msg.data.teamB) teamB.score = msg.data.teamB.score;
                if (msg.data.playerStats) {
                    playerStats = msg.data.playerStats;
                    const hydrated = hydrateTeamsFromServerState(msg.data);
                    if (requireServerRosterHydration && hydrated) {
                        requireServerRosterHydration = false;
                    }
                    // sync the per-player `status` into the team player objects
                    syncPlayerStatusesFromPlayerStats();
                }
                if (msg.data.raidNumber) currentRaidNumber = msg.data.raidNumber;

                // Update empty raid counts from backend (server is source of truth)
                if (msg.data.emptyRaidCounts) {
                    emptyRaidCountA = msg.data.emptyRaidCounts.teamA;
                    emptyRaidCountB = msg.data.emptyRaidCounts.teamB;
                }

                if (msg.data.tossWinner) tossWinner = msg.data.tossWinner;
                if (msg.data.tossDecision) tossDecision = msg.data.tossDecision;
                if (msg.data.firstRaidingTeam) firstRaidingTeam = msg.data.firstRaidingTeam;

                updateDisplay();
                updateRaidInfoUI();
                updateTossInfoUI();
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

function showScorerLockNotice(message) {
    try {
        let banner = document.getElementById('scorer-lock-notice');
        if (!banner) {
            banner = document.createElement('div');
            banner.id = 'scorer-lock-notice';
            banner.style.position = 'fixed';
            banner.style.top = '56px';
            banner.style.right = '12px';
            banner.style.zIndex = '100000';
            banner.style.background = '#b91c1c';
            banner.style.color = '#fff';
            banner.style.padding = '8px 12px';
            banner.style.borderRadius = '8px';
            banner.style.fontWeight = '600';
            banner.style.boxShadow = '0 4px 16px rgba(0,0,0,0.35)';
            document.body.appendChild(banner);
        }
        banner.textContent = message || 'This match is locked by other device.';
    } catch (e) {
        alert(message || 'This match is locked by other device.');
    }
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
            superRaids: 0,
            superTackles: 0,
            totalRaids: 0,
            successfulRaids: 0,
            totalTackles: 0,
            successfulTackles: 0,
            status: player.status,
        };
    });
}

function getDefendingTeam() {
    return getRaidingTeam() === teamA ? teamB : teamA;
}

function getRaidingTeam() {
    const firstTeam = firstRaidingTeam === 'teamB' ? teamB : teamA;
    const secondTeam = firstTeam === teamA ? teamB : teamA;
    return currentRaidNumber % 2 !== 0 ? firstTeam : secondTeam;
}


function handleLobbyTouch(player, isRaiderTouchingLobby) {
    if (!player) return alert("Select a player first.");
    const raidingTeam = getRaidingTeam();
    const defendingTeam = getDefendingTeam();

    const scoringTeam = isRaiderTouchingLobby ? defendingTeam : raidingTeam;

    const lobbyPayload = {
        type: "lobbyTouch",
        data: {
            touchedPlayerId: player.id,
            isRaider: raidingTeam.players.some(p => p.id === player.id),
            scoringTeam: scoringTeam.name === teamA.name ? "A" : "B",
            raiderId: selectedRaider ? selectedRaider.id : null,
            defenderIds: selectedDefenders.map(d => d.id),
            raidNumber: currentRaidNumber
        }
    };

    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(lobbyPayload));
    } else {
        alert('Socket not connected');
    }
}

function normalizeIdValue(value) {
    if (!value) return '';
    if (typeof value === 'string') return value;
    if (typeof value === 'object') {
        if (value.id) return String(value.id);
        if (value._id) return String(value._id);
        if (value.$oid) return String(value.$oid);
    }
    return String(value);
}

async function validateNoTieForKnockoutStage(params, token) {
    const tournamentId = params.get('tournament_id');
    const fixtureId = params.get('fixture_id');
    const championshipId = params.get('championship_id');
    const championshipFixtureId = params.get('championship_fixture_id');

    const authHeaders = token ? { 'Authorization': `Bearer ${token}` } : {};

    if (tournamentId && fixtureId) {
        const res = await fetch(`/api/tournaments/${encodeURIComponent(tournamentId)}/fixtures`, { headers: authHeaders });
        if (res.ok) {
            const fixtures = await res.json();
            if (Array.isArray(fixtures)) {
                const fixture = fixtures.find(f => normalizeIdValue(f.id) === fixtureId);
                const matchType = String(fixture?.matchType || '').toLowerCase();
                if (matchType === 'semifinal' || matchType === 'final') {
                    return {
                        blocked: true,
                        message: `Tie is not allowed in tournament ${matchType}. Please continue match until winner is decided.`
                    };
                }
            }
        }
    }

    if (championshipId && championshipFixtureId) {
        const [championshipRes, fixturesRes] = await Promise.all([
            fetch(`/api/championships/${encodeURIComponent(championshipId)}`, { headers: authHeaders }),
            fetch(`/api/championships/${encodeURIComponent(championshipId)}/fixtures`, { headers: authHeaders })
        ]);

        if (championshipRes.ok && fixturesRes.ok) {
            const championship = await championshipRes.json();
            const fixtures = await fixturesRes.json();
            if (championship && Array.isArray(fixtures)) {
                const fixture = fixtures.find(f => normalizeIdValue(f.id) === championshipFixtureId);
                const totalRounds = Number(championship.totalRounds || 0);
                const roundNumber = Number(fixture?.roundNumber || 0);
                if (totalRounds > 0 && roundNumber >= totalRounds - 1) {
                    return {
                        blocked: true,
                        message: 'Tie is not allowed in championship semifinal/final. Please continue match until winner is decided.'
                    };
                }
            }
        }
    }

    return { blocked: false };
}

async function endGame() {
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

    const queryParams = new URLSearchParams(window.location.search);

    if (teamA.score === teamB.score) {
        try {
            const tieValidation = await validateNoTieForKnockoutStage(queryParams, token);
            if (tieValidation?.blocked) {
                alert(tieValidation.message || 'Tie is not allowed at this knockout stage.');
                return;
            }
        } catch (validationError) {
            console.warn('Tie pre-validation skipped due to fetch error:', validationError);
        }
    }

    // Clear stored match id so refresh won't attempt to rejoin a finished match
    try { localStorage.removeItem(MATCH_STORAGE_KEY); } catch (e) { /* ignore */ }

    // Send API request to end game with authentication
    const eventId = queryParams.get('event_id');
    const tournamentId = queryParams.get('tournament_id');
    const fixtureId = queryParams.get('fixture_id');
    const championshipId = queryParams.get('championship_id');
    const championshipFixtureId = queryParams.get('championship_fixture_id');
    const eventParam = eventId ? `&event_id=${encodeURIComponent(eventId)}` : '';
    const tournamentParam = tournamentId ? `&tournament_id=${encodeURIComponent(tournamentId)}` : '';
    const fixtureParam = fixtureId ? `&fixture_id=${encodeURIComponent(fixtureId)}` : '';
    const championshipParam = championshipId ? `&championship_id=${encodeURIComponent(championshipId)}` : '';
    const championshipFixtureParam = championshipFixtureId ? `&championship_fixture_id=${encodeURIComponent(championshipFixtureId)}` : '';
    try {
        const response = await fetch(`/api/endgame?match_id=${encodeURIComponent(matchId)}${eventParam}${tournamentParam}${fixtureParam}${championshipParam}${championshipFixtureParam}`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });

        if (!response.ok) {
            const errPayload = await response.json().catch(() => null);
            const errorMsg = (errPayload && errPayload.error) ? errPayload.error : 'Failed to end game';
            throw new Error(errorMsg);
        }

        // Successfully ended game, redirect appropriately
        if (tournamentId) {
            window.location.href = `/organizer/tournament?id=${tournamentId}&token=${encodeURIComponent(token)}`;
        } else if (championshipId) {
            window.location.href = `/organizer/championship?id=${championshipId}&token=${encodeURIComponent(token)}`;
        } else if (eventId) {
            window.location.href = `/organizer/event/${eventId}?token=${encodeURIComponent(token)}`;
        } else {
            window.location.href = `/organizer/events?token=${encodeURIComponent(token)}`;
        }
    } catch (error) {
        console.error('Error ending game:', error);
        alert(error.message || 'Failed to end game. Please try again.');
    }
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

function updateTeamRoleHighlight() {
    const teamASection = document.getElementById('teamA-section');
    const teamBSection = document.getElementById('teamB-section');
    const phaseEl = document.getElementById('raid-phase');
    if (!teamASection || !teamBSection) return;

    teamASection.classList.remove('raiding-team-active', 'defending-team-active');
    teamBSection.classList.remove('raiding-team-active', 'defending-team-active');

    const raidingTeam = getRaidingTeam();
    const defendingTeam = getDefendingTeam();

    if (!selectedRaider) {
        if (phaseEl) {
            phaseEl.textContent = 'Phase: Select Raider';
            phaseEl.classList.remove('phase-defender');
            phaseEl.classList.add('phase-raider');
        }
        if (raidingTeam === teamA) {
            teamASection.classList.add('raiding-team-active');
        } else {
            teamBSection.classList.add('raiding-team-active');
        }
        return;
    }

    if (phaseEl) {
        phaseEl.textContent = 'Phase: Select Defenders / Register Result';
        phaseEl.classList.remove('phase-raider');
        phaseEl.classList.add('phase-defender');
    }

    if (defendingTeam === teamA) {
        teamASection.classList.add('defending-team-active');
    } else {
        teamBSection.classList.add('defending-team-active');
    }
}


/**
 * UI Interaction Handlers
 */
function handlePlayerClick(playerId) {
    if (requireServerRosterHydration || !serverRosterHydrated) {
        alert("Syncing live roster from server. Please wait a moment and try again.");
        return;
    }

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
    updateTeamRoleHighlight();
    renderPlayers(); // Re-render to show selection highlighting
}

function toggleDefenderSelection(player) {
    if (selectedDefenders.some(p => p.id === player.id)) {
        selectedDefenders = selectedDefenders.filter(p => p.id !== player.id);
    } else {
        selectedDefenders.push(player);
    }
    updateTeamRoleHighlight();
    renderPlayers(); // Re-render to update selection highlighting
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
        display.innerHTML = `<strong>${raidingTeam.name}</strong> to raid. Select a raider.`;
    } else {
        const defendersList = selectedDefenders.map(p => p.name).join(", ");
        display.innerHTML = `Raider (<strong>${selectedRaider.name}</strong> from ${raidingTeam.name}), Defended By: ${defendersList || 'No defenders selected'}`;
    }

    updateTeamRoleHighlight();
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
                  btn.classList.add("selected-raider");
                  btn.classList.remove("btn-outline-primary");
                  btn.setAttribute("aria-pressed", "true");
              } else if (selectedDefenders.some(p => p.id === player.id)) {
                  btn.classList.add("selected-defender");
                  btn.classList.remove("btn-outline-primary");
                  btn.setAttribute("aria-pressed", "true");
              } else {
                  btn.setAttribute("aria-pressed", "false");
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

    raidElement.innerHTML = `Raid: <strong>${currentRaidNumber}</strong> | Turn: <strong>${raidingTeam.name}</strong> | Status: <strong>${raidType}</strong>`;
}

function updateTossInfoUI() {
    const el = document.getElementById('toss-info');
    if (!el) return;
    if (!tossWinner) {
        el.style.display = 'none';
        return;
    }
    const winnerName = tossWinner === 'teamB' ? teamB.name : teamA.name;
    const decisionText = tossDecision === 'defend' ? 'Defend First' : 'Raid First';
    el.textContent = `Toss: ${winnerName} | Decided To: ${decisionText}`;
    el.style.display = 'block';
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
    requireServerRosterHydration = resumeFlag;
    serverRosterHydrated = !resumeFlag;
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
            const selectedA = resumeFlag ? null : JSON.parse(localStorage.getItem("teamA_selected"));
            const selectedB = resumeFlag ? null : JSON.parse(localStorage.getItem("teamB_selected"));

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

            // Update toss selection labels with actual team names
            const tossWinnerSelect = document.getElementById('toss-winner');
            if (tossWinnerSelect) {
                const optA = tossWinnerSelect.querySelector('option[value="teamA"]');
                const optB = tossWinnerSelect.querySelector('option[value="teamB"]');
                if (optA) optA.textContent = teamA.name || 'Team A';
                if (optB) optB.textContent = teamB.name || 'Team B';
            }
            
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

                    const tossWinnerSelect = document.getElementById('toss-winner');
                    const tossDecisionSelect = document.getElementById('toss-decision');
                    if (!tossWinnerSelect || !tossWinnerSelect.value) {
                        return alert('Please select which team won the toss');
                    }

                    tossWinner = tossWinnerSelect.value;
                    tossDecision = tossDecisionSelect ? tossDecisionSelect.value : 'raid';
                    firstRaidingTeam = tossDecision === 'raid'
                        ? tossWinner
                        : (tossWinner === 'teamA' ? 'teamB' : 'teamA');

                    currentRaidNumber = 1;
                    updateCurrentRaidDisplay();
                    updateRaidInfoUI();
                    updateTossInfoUI();

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

            // If match_id is provided in URL, prefill it.
            // Only auto-join when resume=1 is explicitly set.
            if (prefillMatchId) {
                matchId = prefillMatchId.trim();
                if (matchInput) matchInput.value = matchId;
                try { localStorage.setItem(MATCH_STORAGE_KEY, matchId); } catch (e) { console.warn('Failed to persist match id', e); }
                if (resumeFlag) {
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
                } else {
                    setMatchIdDisplay('—');
                    const overlay = document.getElementById('match-setup');
                    if (overlay) overlay.style.display = 'flex';
                }
            }

            // If a match id was previously stored, auto-join that match (persistence across refreshes)
            const stored = (() => { try { return localStorage.getItem(MATCH_STORAGE_KEY); } catch (e) { return null; } })();
            if (!autoJoinedFromPrefill && stored && stored.trim() !== '') {
                matchId = stored.trim();
                if (matchInput) matchInput.value = matchId;
                // Only auto-join (and hide overlay) if explicitly allowed via URL (prefill/resume)
                const allowAutoJoin = resumeFlag;
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