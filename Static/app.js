let socket = null;

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
function setupWebSocket() {
    if (socket !== null) {
        console.log("WebSocket already exists");
        return;
    }

    socket = new WebSocket("ws://localhost:3000/ws/scorer");
    console.log("Setting up WebSocket connection...");

    // The 'onopen' handler is set later in DOMContentLoaded after teams are loaded.

    socket.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            if (msg.error) {
                alert(`Server error: ${msg.error}`);
                return;
            }
            if (msg.data) {
                // Update local state based on server response
                if (msg.data.teamA) teamA.score = msg.data.teamA.score;
                if (msg.data.teamB) teamB.score = msg.data.teamB.score;
                if (msg.data.playerStats) {
                    playerStats = msg.data.playerStats;
                    // sync the per-player `status` into the team player objects
                    syncPlayerStatusesFromPlayerStats();
                }
                if (msg.data.raidNumber) currentRaidNumber = msg.data.raidNumber;

                // Check for Empty Raid Counts from server (if applicable)
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
    };
    
    socket.onclose = () => {
        console.log("WebSocket connection closed.");
    };
}

/**
 * Game State Management
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

function canRaid(team) {
    const activePlayers = team.players.filter(player => player.status === "in").length;
    return activePlayers >= 1 && activePlayers <= 7;
}

function enforceLobbyRule() {
    const raidingTeam = getRaidingTeam();
    if (!canRaid(raidingTeam)) {
        alert(`${raidingTeam.name} does not have enough players to raid!`);
        return false;
    }
    return true;
}

function revivePlayers(team, count) {
    let outPlayers = team.players.filter(p => p.status === "out");
    for (let i = 0; i < count && i < outPlayers.length; i++) {
        outPlayers[i].status = "in";
    }
}

function checkAllOut() {
    [teamA, teamB].forEach(team => {
        if (team.players.every(p => p.status === "out")) {
            team.players.forEach(p => p.status = "in");
            const opponent = team === teamA ? teamB : teamA;
            opponent.score += 2;
        }
    });
}

function handleLobbyTouch(player, isRaiderTouchingLobby) {
    if (!player) return alert("Select a player first.");
    
    const raidingTeam = getRaidingTeam();
    const defendingTeam = getDefendingTeam();

    // Determine the team that gets the point
    const scoringTeam = isRaiderTouchingLobby ? defendingTeam : raidingTeam;
    
    // Create the payload for the server
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

    if (teamA.score > teamB.score) {
        message = `${teamA.name} wins`;
    } else if (teamA.score < teamB.score) {
        message = `${teamB.name} wins`;
    } else {
        message = "It was a tie";
    }

    alert(message);
    window.location.href = `/endgame`;
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

function resetEmptyRaidCount(team) {
    if (team === teamA) emptyRaidCountA = 0;
    else emptyRaidCountB = 0;
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
 * Score Action Handlers (Sending data to server)
 */
function raidSuccessful() {
    if (!selectedRaider) {
        alert("Select a raider.");
        return;
    }

    const payload = {
        raidType: "successful",
        raiderId: selectedRaider.id,
        defenderIds: selectedDefenders.map(d => d.id),
        raidingTeam: currentRaidNumber % 2 !== 0 ? "A" : "B",
        bonusTaken: bonusTaken,
        emptyRaidCounts: {
            teamA: emptyRaidCountA,
            teamB: emptyRaidCountB
        }
    };

    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(payload));
        resetEmptyRaidCount(getRaidingTeam()); // Reset count on successful raid
    } else {
        alert('Socket not connected');
    }
}


function defenseSuccessful() {
    if (!selectedRaider) {
        alert("Select a raider.");
        return;
    }

    const payload = {
        raidType: "defense",
        raiderId: selectedRaider.id,
        defenderIds: selectedDefenders.map(d => d.id),
        raidingTeam: currentRaidNumber % 2 !== 0 ? "A" : "B",
        bonusTaken: bonusTaken,
        emptyRaidCounts: {
            teamA: emptyRaidCountA,
            teamB: emptyRaidCountB
        }
    };
    if (socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify(payload));
        resetEmptyRaidCount(getRaidingTeam()); // Reset count on successful defense
    }
}

function emptyRaid() {
    if (!selectedRaider) {
        alert("Select a raider.");
        return;
    }

    const isTeamA = currentRaidNumber % 2 !== 0;
    if (isTeamA) {
        emptyRaidCountA++;
    } else {
        emptyRaidCountB++;
    }

    const payload = {
        raidType: "empty",
        raiderId: selectedRaider.id,
        defenderIds: [],
        raidingTeam: isTeamA ? "A" : "B",
        bonusTaken: bonusTaken,
        emptyRaidCounts: {
            teamA: emptyRaidCountA,
            teamB: emptyRaidCountB
        }
    };
    if (socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify(payload));
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

// Sync statuses from the authoritative playerStats map (received from server)
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
        // If a currently selected raider was marked out, clear selection
        if (selectedRaider && playerStats[selectedRaider.id] && playerStats[selectedRaider.id].status !== 'in') {
            selectedRaider = null;
        }
        // Remove any selected defenders who are now out
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
    const params = new URLSearchParams(window.location.search);
    const team1Id = params.get("team1_id");
    const team2Id = params.get("team2_id");

    console.log("DOM fully loaded. Starting initialization.");

    if (team1Id && team2Id) {
        try {
            // 1. Fetch Team Data
            const [res1, res2] = await Promise.all([
                fetch(`/api/team/${team1Id}`),
                fetch(`/api/team/${team2Id}`)
            ]);

            const data1 = await res1.json();
            const data2 = await res2.json();

            // Load selected players from localStorage
            const selectedA = JSON.parse(localStorage.getItem("teamA_selected"));
            const selectedB = JSON.parse(localStorage.getItem("teamB_selected"));

            // 2. Initialize Teams and Players
            teamA.name = data1.team_name;
            // Ensure players array contains the required 'status' field
            teamA.players = selectedA ? selectedA.map(p => ({ ...p, status: "in" })) : [];
            teamB.name = data2.team_name;
            teamB.players = selectedB ? selectedB.map(p => ({ ...p, status: "in" })) : [];

            // 3. Initialize Player Stats
            initializePlayerStats(teamA);
            initializePlayerStats(teamB);

            // 4. Setup WebSocket Connection
            setupWebSocket();

            // 5. Define onopen handler (Guaranteed to have team data now)
            // Do NOT send full local initial state to server here. The server is
            // authoritative and will push the saved `gameStats` from Redis to the
            // client on connect. This avoids overwriting server state when the
            // front-end refreshes.
            socket.onopen = () => {
                console.log("WebSocket connection established (awaiting server state)");
                // UI will be updated when the server sends the authoritative state
            };

        } catch (err) {
            console.error("Failed to load teams:", err);
            // Alert user if data loading fails
            alert("Error: Failed to load team data. Check console for details.");
        }
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