const ws = new WebSocket("ws://localhost:3000/ws/viewer");

ws.onopen = () => {
    console.log("Connected to WebSocket server.");
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);

    if (data.type === "gameStats") {
        // Update scores
        document.getElementById("teamA-name").textContent = data.data.teamA.name;
        document.getElementById("teamA-score").textContent = data.data.teamA.score;
        document.getElementById("teamB-name").textContent = data.data.teamB.name;
        document.getElementById("teamB-score").textContent = data.data.teamB.score;

        // Update commentary
        const raid = data.data.raidDetails;
        const commentary = `Raid by ${raid.raider}: ${raid.pointsGained} points ${
            raid.bonusTaken ? "(Bonus taken)" : ""
        } ${raid.superTackle ? "(Super Tackle)" : ""}`;
        document.getElementById("live-commentary").textContent = commentary;
    }
};

ws.onclose = () => {
    console.log("Disconnected from WebSocket server.");
};
