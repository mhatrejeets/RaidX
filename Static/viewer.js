// Viewer client that can join a match via URL param or manual input
let ws = null;
let matchId = null;

function joinMatch(id) {
    if (!id) return;
    matchId = id;
    // hide join UI
    const joinDiv = document.getElementById('viewer-join');
    if (joinDiv) joinDiv.style.display = 'none';

    // Use current hostname for WebSocket connection
    const wsProto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProto}//${location.host}/ws/viewer`;
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log("Connected to WebSocket server.");
        ws.send(JSON.stringify({ type: 'join', matchId }));
        const info = document.getElementById('viewer-info');
        if (info) info.textContent = `Viewing Match: ${matchId}`;
        const mid = document.getElementById('viewer-matchid'); if (mid) mid.textContent = `Match: ${matchId}`;
        const conn = document.getElementById('viewer-conn'); if (conn) { conn.textContent = 'Connected'; conn.style.background = '#10b981'; }
    };

    ws.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            if (data.type === "gameStats" || data.data) {
                const payload = data.data || data;
                if (payload.teamA) document.getElementById("teamA-name").textContent = payload.teamA.name;
                if (payload.teamA) document.getElementById("teamA-score").textContent = payload.teamA.score;
                if (payload.teamB) document.getElementById("teamB-name").textContent = payload.teamB.name;
                if (payload.teamB) document.getElementById("teamB-score").textContent = payload.teamB.score;

                // commentary if present
                const raid = payload.raidDetails || payload.raid || {};
                const commentary = raid.raider ? `Raid by ${raid.raider}: ${raid.pointsGained || 0} points ${raid.bonusTaken ? "(Bonus taken)" : ""} ${raid.superTackle ? "(Super Tackle)" : ""}` : 'Waiting for match updates...';
                const liveEl = document.getElementById("live-commentary");
                if (liveEl) liveEl.textContent = commentary;
            }
        } catch (e) { console.error('Invalid WS message', e); }
    };

    ws.onclose = () => { console.log("Disconnected from WebSocket server."); const conn = document.getElementById('viewer-conn'); if (conn) { conn.textContent = 'Disconnected'; conn.style.background = '#6b7280'; } };
}

document.addEventListener('DOMContentLoaded', () => {
    const params = new URLSearchParams(window.location.search);
    const urlMatch = params.get('match_id');
    const joinInput = document.getElementById('viewer-match-id-input');
    const joinBtn = document.getElementById('viewer-join-btn');
    const info = document.getElementById('viewer-info');

    if (urlMatch) {
        if (joinInput) joinInput.value = urlMatch;
        joinMatch(urlMatch);
    } else {
        // show join panel
        if (joinBtn) {
            joinBtn.addEventListener('click', () => {
                const v = joinInput ? joinInput.value.trim() : '';
                if (!v) return alert('Enter a match id to join');
                joinMatch(v);
            });
        }
    }
});
