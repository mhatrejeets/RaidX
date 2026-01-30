// State variables
let ws = null;
let matchId = null;
// Optional token (viewer is allowed without auth)
let jwtToken = (typeof getValidToken === 'function') ? getValidToken() : null;
let matchEnded = false;

function joinMatch(id) {
    if (!id) return;
    
    matchId = id;
    // hide join UI
    const joinDiv = document.getElementById('viewer-join');
    if (joinDiv) joinDiv.style.display = 'none';

    // Use current hostname for WebSocket connection; token is optional
    const wsProto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = jwtToken
        ? `${wsProto}//${location.host}/ws/viewer?token=${encodeURIComponent(jwtToken)}`
        : `${wsProto}//${location.host}/ws/viewer`;
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
            
            // Check for error indicating match ended or not found
            if (data.error && (data.error.includes('not initialized') || data.error.includes('not found') || data.error.includes('ended'))) {
                matchEnded = true;
                ws.close();
                showEndedMatchUI();
                return;
            }
            
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
    };
}

function showEndedMatchUI() {
    if (matchEnded) {
        // Ensure ended UI is visible and wired once
        const endedSection = document.getElementById('viewer-ended');
        if (endedSection) endedSection.style.display = 'block';
        const viewBtn = document.getElementById('view-match-score-btn');
        if (viewBtn && !viewBtn.dataset.bound) {
            viewBtn.addEventListener('click', fetchMatchScore);
            viewBtn.dataset.bound = 'true';
        }
    }

    // Hide live commentary section
    const commentaryDiv = document.querySelector('.commentary');
    if (commentaryDiv) commentaryDiv.style.display = 'none';
    
    // Update connection status
    const conn = document.getElementById('viewer-conn');
    if (conn) {
        conn.textContent = 'Match Ended';
        conn.style.background = '#f59e0b';
    }
    
    // Update commentary to show match ended message
    const liveEl = document.getElementById('live-commentary');
    if (liveEl) {
        liveEl.innerHTML = '<strong style="color:#f59e0b;">Match Ended</strong><br>This match has been completed and archived.';
    }
    
    // Show the match ended UI (pre-rendered in HTML)
    const endedSection = document.getElementById('viewer-ended');
    if (endedSection) endedSection.style.display = 'block';

    const viewBtn = document.getElementById('view-match-score-btn');
    if (viewBtn && !viewBtn.dataset.bound) {
        viewBtn.addEventListener('click', fetchMatchScore);
        viewBtn.dataset.bound = 'true';
    }

    // Try to fetch and show the final score from MongoDB
    tryFetchFinalScore();
}

function tryFetchFinalScore() {
    if (!matchId) return;
    
    // Try to get match details from MongoDB via API
    fetch(`/api/match/${matchId}`)
        .then(res => {
            if (!res.ok) throw new Error('Match not found');
            return res.json();
        })
        .then(match => {
            if (match && match.data) {
                const data = match.data;
                if (data.teamA) {
                    document.getElementById('teamA-name').textContent = data.teamA.name || 'Team A';
                    document.getElementById('teamA-score').textContent = data.teamA.score || 0;
                }
                if (data.teamB) {
                    document.getElementById('teamB-name').textContent = data.teamB.name || 'Team B';
                    document.getElementById('teamB-score').textContent = data.teamB.score || 0;
                }
            }
        })
        .catch(err => {
            console.log('Could not fetch final score:', err);
        });
}

function fetchMatchScore() {
    if (!matchId) {
        alert('No match ID available');
        return;
    }
    
    // Navigate to the match details page
    const token = jwtToken || (typeof getValidToken === 'function' ? getValidToken() : null);
    if (token) {
        window.location.href = `/matches/${matchId}?token=${encodeURIComponent(token)}`;
    } else {
        // Try without token; backend will redirect to login if needed
        window.location.href = `/matches/${matchId}`;
    }
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
