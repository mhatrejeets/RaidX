let team1Players = [];
let team2Players = [];
let selectedTeam1 = [];
let selectedTeam2 = [];

document.addEventListener("DOMContentLoaded", async () => {
  const params = new URLSearchParams(window.location.search);
  const team1Id = params.get("team1_id");
  const team2Id = params.get("team2_id");

  if (!team1Id || !team2Id) return;

  const token = localStorage.getItem('token');
  const fetchOptions = {
    headers: {
      'Authorization': `Bearer ${token}`
    }
  };
  
  const [res1, res2] = await Promise.all([
    fetch(`/api/team/${team1Id}`, fetchOptions),
    fetch(`/api/team/${team2Id}`, fetchOptions)
  ]);

  const team1 = await res1.json();
  const team2 = await res2.json();

  document.getElementById("team1-name").textContent = team1.team_name;
  document.getElementById("team2-name").textContent = team2.team_name;

  team1Players = team1.players;
  team2Players = team2.players;

  renderTeamPlayers("team1-players", team1Players, selectedTeam1, updateButtonState);
  renderTeamPlayers("team2-players", team2Players, selectedTeam2, updateButtonState);

  document.getElementById("start-match").addEventListener("click", () => {
    // Store selected players in localStorage
    localStorage.setItem("teamA_selected", JSON.stringify(selectedTeam1));
    localStorage.setItem("teamB_selected", JSON.stringify(selectedTeam2));

    // Redirect to scorer
    window.location.href = `/scorer?team1_id=${team1Id}&team2_id=${team2Id}&team1_name=${team1.team_name}&team2_name=${team2.team_name}`;
  });
});

function renderTeamPlayers(containerId, players, selectedList, updateCallback) {
  const container = document.getElementById(containerId);
  container.innerHTML = "";

  players.forEach(player => {
    const btn = document.createElement("button");
    btn.textContent = player.name;
    btn.className = "player-button";
    btn.addEventListener("click", () => {
      const isSelected = selectedList.some(p => p.id === player.id);
      if (isSelected) {
        selectedList.splice(selectedList.findIndex(p => p.id === player.id), 1);
        btn.classList.remove("selected");
      } else if (selectedList.length < 7) {
        selectedList.push(player);
        btn.classList.add("selected");
      }
      updateCallback();
    });
    container.appendChild(btn);
  });
}

function updateButtonState() {
  const btn = document.getElementById("start-match");
  btn.disabled = !(selectedTeam1.length === 7 && selectedTeam2.length === 7);
}
