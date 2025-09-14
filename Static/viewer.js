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

        // Update commentary list
            // Commentary is now handled by the commentary list below
            const commentaryList = data.extra && data.extra.commentaryList ? data.extra.commentaryList : [];
            const commentaryDiv = document.getElementById("live-commentary-list");
            if (commentaryList.length > 0) {
                commentaryDiv.innerHTML = commentaryList.map(comment => `<p>${comment}</p>`).join("");
            } else {
                commentaryDiv.innerHTML = `<p>Waiting for match updates...</p>`;
            }
    }
};

ws.onclose = () => {
    console.log("Disconnected from WebSocket server.");
};
