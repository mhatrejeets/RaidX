// Constants
const JWT_STORAGE_KEY = 'jwtToken';

// State variables
let ws = null;
let matchId = null;
let jwtToken = localStorage.getItem(JWT_STORAGE_KEY);

function joinMatch(id) {
    if (!id) return;
    if (!jwtToken) {
        const currentUrl = encodeURIComponent(window.location.href);
        window.location.href = `/login?returnUrl=${currentUrl}`;
        return;
    }
    
    matchId = id;
    // hide join UI
    const joinDiv = document.getElementById('viewer-join');
    if (joinDiv) joinDiv.style.display = 'none';

    // Use current hostname for WebSocket connection with JWT token
    const wsProto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProto}//${location.host}/ws/viewer?token=${jwtToken}`;
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

                // If match ended, show winner
                if (payload.matchEnded) {
                    let winner = "";
                    if (payload.teamA && payload.teamB) {
                        if (payload.teamA.score > payload.teamB.score) {
                            winner = `${payload.teamA.name} wins!`;
                        } else if (payload.teamB.score > payload.teamA.score) {
                            winner = `${payload.teamB.name} wins!`;
                        } else {
                            winner = "Match Draw!";
                        }
                    }
                    const liveEl = document.getElementById("live-commentary");
                    if (liveEl) liveEl.textContent = `Match Ended. ${winner}`;
                    return;
                }

                // commentary if present
                const raid = payload.raidDetails || payload.raid || {};
                const commentary = raid.raider ? `Raid by ${raid.raider}: ${raid.pointsGained || 0} points ${raid.bonusTaken ? "(Bonus taken)" : ""} ${raid.superTackle ? "(Super Tackle)" : ""}` : 'Waiting for match updates...';
                const liveEl = document.getElementById("live-commentary");
                if (liveEl) liveEl.textContent = commentary;
            }
        } catch (e) { console.error('Invalid WS message', e); }
    };

    ws.onclose = () => { 
        console.log("Disconnected from WebSocket server."); 
        const conn = document.getElementById('viewer-conn'); 
        if (conn) { 
            conn.textContent = 'Disconnected'; 
            conn.style.background = '#6b7280'; 
        } 
    };

    ws.onerror = (error) => {
        console.error("WebSocket error:", error);
        const conn = document.getElementById('viewer-conn'); 
        if (conn) { 
            conn.textContent = 'Error'; 
            conn.style.background = '#ef4444'; 
        }
        if (!jwtToken) {
            const currentUrl = encodeURIComponent(window.location.href);
            window.location.href = `/login?returnUrl=${currentUrl}`;
        }
    };
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
    }

    // Always show join panel if joinBtn exists
    if (joinBtn) {
        joinBtn.addEventListener('click', () => {
            const v = joinInput ? joinInput.value.trim() : '';
            if (!v) return alert('Enter a match id to join');
            joinMatch(v);
        });
    }
});
