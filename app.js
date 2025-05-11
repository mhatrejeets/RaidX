let teamA = {
  name: "Team A",
  score: 0,
  players: [
    { id: "A1", name: "Player A1", status: "in" },
    { id: "A2", name: "Player A2", status: "in" },
    { id: "A3", name: "Player A3", status: "in" },
    { id: "A4", name: "Player A4", status: "in" },
    { id: "A5", name: "Player A5", status: "in" },
    { id: "A6", name: "Player A6", status: "in" },
    { id: "A7", name: "Player A7", status: "in" },
  ],
};

let teamB = {
  name: "Team B",
  score: 0,
  players: [
    { id: "B1", name: "Player B1", status: "in" },
    { id: "B2", name: "Player B2", status: "in" },
    { id: "B3", name: "Player B3", status: "in" },
    { id: "B4", name: "Player B4", status: "in" },
    { id: "B5", name: "Player B5", status: "in" },
    { id: "B6", name: "Player B6", status: "in" },
    { id: "B7", name: "Player B7", status: "in" },
  ],
};

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

// function handleLobbyTouch(player, isRaiderTouchingLobby) {
//   const raidingTeam = getRaidingTeam();
//   const defendingTeam = getDefendingTeam();

//   if (isRaiderTouchingLobby) {
//     defendingTeam.score += 1;
//     alert(`${raidingTeam.name}'s raider touched the lobby! ${defendingTeam.name} gets 1 point.`);
//   } else {
//     raidingTeam.score += 1;
//     alert(`${defendingTeam.name}'s defender touched the lobby! ${raidingTeam.name} gets 1 point.`);
//   }

//   nextRaid();
// }
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
  alert("Game Ended");
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

  raidingTeam.score += selectedDefenders.length;

  if (bonusTaken) raidingTeam.score += 1;

  selectedDefenders.forEach(d => d.status = "out");

  revivePlayers(raidingTeam, selectedDefenders.length);

  if (defendingTeam.players.filter(p => p.status === "in").length <= 3) {
    defendingTeam.score += 1;
  }

  resetEmptyRaidCount(raidingTeam);
  checkAllOut();
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

  if (defendingTeam.players.filter(p => p.status === "in").length <= 3) points += 1;


  selectedRaider.status = "out";
  defendingTeam.score += points;
  if (bonusTaken) raidingTeam.score += 1;

  revivePlayers(defendingTeam, 1);
  resetEmptyRaidCount(raidingTeam);
  checkAllOut();
  nextRaid();
}

function emptyRaid() {
  if (!selectedRaider) {
    alert("Select a raider.");
    return;
  }

  const raidingTeam = getRaidingTeam();
  const defendingTeam = getDefendingTeam();

  if (bonusTaken) raidingTeam.score += 1;

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
    alert("Do or Die Raid! Raider is out and defending team gets 1 point.");
    if (isTeamA) emptyRaidCountA = 0;
    else emptyRaidCountB = 0;
  }

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

// function renderPlayers() {
//   const render = (team, containerId) => {
//     const container = document.getElementById(containerId);
//     container.innerHTML = "";
//     team.players.forEach(player => {
//       const btn = document.createElement("button");
//       btn.className = `player-card btn ${player.status === "in" ? "btn-outline-primary" : "btn-secondary"}`;
//       btn.textContent = player.name;
//       btn.onclick = () => handlePlayerClick(player.id);
//       container.appendChild(btn);
//     });
//   };

//   render(teamA, "teamA-players");
//   render(teamB, "teamB-players");
// }

function updateBonusToggleVisibility() {
  const bonusToggle = document.getElementById("bonus-toggle");
  const opposingTeam = getDefendingTeam();
  const inPlayers = opposingTeam.players.filter(p => p.status === "in").length;

  bonusToggle.disabled = inPlayers < 6;
}

function updateRaidInfoUI() {
  document.getElementById("raid-number").textContent = `Raid: ${raid}`;
}

document.addEventListener("DOMContentLoaded", () => {
  document.getElementById("teamA-name").textContent = teamA.name;
  document.getElementById("teamB-name").textContent = teamB.name;
  document.getElementById("teamA-score").textContent = teamA.score;
  document.getElementById("teamB-score").textContent = teamB.score;
  document.getElementById("teamA-header").textContent = teamA.name;
  document.getElementById("teamB-header").textContent = teamB.name;

  document.getElementById("bonus-toggle").addEventListener("change", () => {
    bonusTaken = document.getElementById("bonus-toggle").checked;
  });

  renderPlayers();
  updateBonusToggleVisibility();
  updateRaidInfoUI();
});
