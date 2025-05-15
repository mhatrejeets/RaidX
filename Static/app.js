const socket = new WebSocket("ws://localhost:3000/ws/scorer"); // Adjust your port/path if needed

socket.onopen = () => {
  console.log("WebSocket connection established");
};

socket.onerror = (error) => {
  console.error("WebSocket error:", error);
};

function sendEnhancedStats(raidDetails) {
  if (socket.readyState === WebSocket.OPEN) {
    const enhancedStats = {
      type: "gameStats",
      data: {
        teamA: {
          name: teamA.name,
          score: teamA.score,
        },
        teamB: {
          name: teamB.name,
          score: teamB.score,
        },
        playerStats: playerStats,
        raidDetails: raidDetails, // Include the detailed raid information
      },
    };
    socket.send(JSON.stringify(enhancedStats));
  }
}



let teamA = { name: "", score: 0, players: [] };
let teamB = { name: "", score: 0, players: [] };

let playerStats = {};

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

let game = true;
let raid = 1;
let selectedRaider = null;
let selectedDefenders = [];
let bonusTaken = false;
let emptyRaidCountA = 0;
let emptyRaidCountB = 0;
let isDoOrDieRaid = false;

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


function handleLobbyTouch(player, isRaiderTouchingLobby) {
  const raidingTeam = getRaidingTeam();
  const defendingTeam = getDefendingTeam();

  if (isRaiderTouchingLobby) {
    // Raider touches lobby – out
    defendingTeam.score += 1;
    player.status = "out";  // Mark raider as out
    alert(`${raidingTeam.name}'s raider touched the lobby! ${defendingTeam.name} gets 1 point.`);
  } else {
    // Defender touches lobby – out
    raidingTeam.score += 1;
    player.status = "out";  // Mark defender as out
    alert(`${defendingTeam.name}'s defender touched the lobby! ${raidingTeam.name} gets 1 point.`);
  }

  checkAllOut(); // In case this causes an all-out
  renderPlayers(); // Update UI to reflect out status
  nextRaid();
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

  // Redirect after the alert is dismissed
  setTimeout(function() {
    window.location.href = "/endgame";
  }, 100); // small delay to allow alert to complete
}


function handlePlayerClick(playerId) {
  const currentTeam = getRaidingTeam();
  const opposingTeam = getDefendingTeam();

  let player = [...currentTeam.players, ...opposingTeam.players].find(p => p.id === playerId);

  if (!player || player.status !== "in") return;

  if (currentTeam.players.includes(player)) {
    selectedRaider = player;
  } else {
    toggleDefenderSelection(player);
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

function updateCurrentRaidDisplay() {
  let display = document.getElementById("current-raid");
  if (!selectedRaider) {
    display.textContent = "Select a raider.";
  } else {
    display.textContent = `Raider: ${selectedRaider.name}, Defenders: ${selectedDefenders.map(p => p.name).join(", ")}`;
  }
}

function getDefendingTeam() {
  return raid % 2 !== 0 ? teamB : teamA;
}

function getRaidingTeam() {
  return raid % 2 !== 0 ? teamA : teamB;
}

function raidSuccessful() {
  if (!selectedRaider || selectedDefenders.length === 0) {
    alert("Select a raider and at least one defender.");
    return;
  }

  const raidingTeam = getRaidingTeam();
  const defendingTeam = getDefendingTeam();
  let raidPoints = selectedDefenders.length;

  // Update scores and stats
  raidingTeam.score += raidPoints;
  playerStats[selectedRaider.id].raidPoints += raidPoints;
  playerStats[selectedRaider.id].totalPoints += raidPoints;

  let eliminatedDefenders = [];
  selectedDefenders.forEach((def) => {
    def.status = "out";
    eliminatedDefenders.push(def.name);
  });

  let raidDetails = {
    type: "raidSuccess",
    raider: selectedRaider.name,
    defendersEliminated: eliminatedDefenders,
    pointsGained: raidPoints,
    bonusTaken: bonusTaken,
  };

  if (bonusTaken) {
    raidingTeam.score += 1;
    playerStats[selectedRaider.id].raidPoints += 1;
    playerStats[selectedRaider.id].totalPoints += 1;
    raidDetails.bonusTaken = true;
  }

  revivePlayers(raidingTeam, selectedDefenders.length);
  resetEmptyRaidCount(raidingTeam);
  checkAllOut();
  sendEnhancedStats(raidDetails);
  nextRaid();
}


function defenseSuccessful() {
  if (!selectedRaider) {
    alert("Select a raider.");
    return;
  }

  const raidingTeam = getRaidingTeam();
  const defendingTeam = getDefendingTeam();

  let points = 1;
  if (defendingTeam.players.filter((p) => p.status === "in").length <= 3) {
    points += 1; // Super tackle
  }

  selectedRaider.status = "out";
  defendingTeam.score += points;

  let defendingPlayers = selectedDefenders.map((def) => def.name);
  selectedDefenders.forEach((def) => {
    playerStats[def.id].defencePoints += 1;
    playerStats[def.id].totalPoints += 1;
  });

  let raidDetails = {
    type: "defenseSuccess",
    raider: selectedRaider.name,
    defenders: defendingPlayers,
    pointsGained: points,
    superTackle: points > 1,
  };

  revivePlayers(defendingTeam, 1);
  resetEmptyRaidCount(raidingTeam);
  checkAllOut();
  sendEnhancedStats(raidDetails);
  nextRaid();
}

function emptyRaid() {
  if (!selectedRaider) {
    alert("Select a raider.");
    return;
  }

  const raidingTeam = getRaidingTeam();
  const defendingTeam = getDefendingTeam();

  let raidDetails = {
    type: "emptyRaid",
    raider: selectedRaider.name,
    bonusTaken: bonusTaken,
  };

  if (bonusTaken) {
    raidingTeam.score += 1;
    playerStats[selectedRaider.id].raidPoints += 1;
    playerStats[selectedRaider.id].totalPoints += 1;
    raidDetails.bonusTaken = true;
  }

  const isTeamA = raidingTeam === teamA;
  if (isTeamA) {
    emptyRaidCountA++;
    isDoOrDieRaid = emptyRaidCountA >= 3;
  } else {
    emptyRaidCountB++;
    isDoOrDieRaid = emptyRaidCountB >= 3;
  }

  if (isDoOrDieRaid) {
    selectedRaider.status = "out";
    defendingTeam.score += 1;
    raidDetails.type = "doOrDieRaid";
    emptyRaidCountA = emptyRaidCountB = 0; // Reset counters
  }

  sendEnhancedStats(raidDetails);
  nextRaid();
}

function resetEmptyRaidCount(team) {
  if (team === teamA) emptyRaidCountA = 0;
  else emptyRaidCountB = 0;
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

document.getElementById("raider-lobby-entry").addEventListener("click", function () {
  handleLobbyTouch(selectedRaider, true);
});

document.getElementById("defender-lobby-entry").addEventListener("click", function () {
  if (selectedDefenders.length > 0) {
    handleLobbyTouch(selectedDefenders[0], false);
  }
});

function nextRaid() {
  selectedRaider = null;
  selectedDefenders = [];

  bonusTaken = false;
  isDoOrDieRaid = false;

  document.getElementById("bonus-toggle").checked = false;
  document.getElementById("bonus-toggle").disabled = true;


  raid++;
  updateDisplay();
  updateCurrentRaidDisplay();
  updateBonusToggleVisibility();
  updateRaidInfoUI();
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

      // Add basic classes
      btn.className = `player-card btn`;

      if (player.status === "in") {
        btn.classList.add("btn-outline-primary");
      } else {
        btn.classList.add("btn-secondary");
        btn.style.textDecoration = "line-through"; // Strike-through effect
        btn.disabled = true; // Prevent selection of out players
        btn.style.opacity = "0.6"; // Make it look faded
      }

      btn.textContent = player.name;
      btn.onclick = () => handlePlayerClick(player.id);

      container.appendChild(btn);
    });
  };

  render(teamA, "teamA-players");
  render(teamB, "teamB-players");
}
function updateBonusToggleVisibility() {
  const bonusToggle = document.getElementById("bonus-toggle");
  const opposingTeam = getDefendingTeam();
  const inPlayers = opposingTeam.players.filter(p => p.status === "in").length;

  bonusToggle.disabled = inPlayers < 6;
}

function updateRaidInfoUI() {
  if (document.readyState === "loading") {
    console.warn("⚠️ DOM not fully loaded");
    
    return;
  }

  const raidElement = document.getElementById("raid-number");

  if (!raidElement) {
    console.warn("⚠️ 'raid-number' element not found");
    
    return;
  }

  raidElement.textContent = `Raid: ${raid}`;
}





document.addEventListener("DOMContentLoaded", async () => {
  const params = new URLSearchParams(window.location.search);
  const team1Id = params.get("team1_id");
  const team2Id = params.get("team2_id");
  const team1Name = params.get("team1_name");
  const team2Name = params.get("team2_name");
  const raidElement = document.getElementById("raid-number");
  console.log("DOM fully loaded. Raid Element:", raidElement);

  socket.onopen = () => console.log("WebSocket connection established");
  socket.onerror = (err) => console.error("WebSocket error:", err);
  socket.onclose = () => console.log("WebSocket closed");

  if (team1Id && team2Id) {
    try {
      const [res1, res2] = await Promise.all([
        fetch(`/api/team/${team1Id}`),
        fetch(`/api/team/${team2Id}`)
      ]);

      const data1 = await res1.json();
      const data2 = await res2.json();

      const selectedA = JSON.parse(localStorage.getItem("teamA_selected"));
      const selectedB = JSON.parse(localStorage.getItem("teamB_selected"));

      teamA.name = data1.team_name;
      teamA.players = selectedA.map(p => ({ ...p, status: "in" }));
      teamB.name = data2.team_name;
      teamB.players = selectedB.map(p => ({ ...p, status: "in" }));



    } catch (err) {
      console.error("Failed to load teams:", err);
    }
  }

  // Update UI with names
  document.getElementById("teamA-name").textContent = teamA.name;
  document.getElementById("teamB-name").textContent = teamB.name;
  document.getElementById("teamA-header").textContent = teamA.name;
  document.getElementById("teamB-header").textContent = teamB.name;

  document.getElementById("teamA-score").textContent = teamA.score;
  document.getElementById("teamB-score").textContent = teamB.score;


    initializePlayerStats(teamA);
    initializePlayerStats(teamB);

  document.getElementById("bonus-toggle").addEventListener("change", () => {
    bonusTaken = document.getElementById("bonus-toggle").checked;
  });

  renderPlayers();
  updateBonusToggleVisibility();
  updateRaidInfoUI();
});
