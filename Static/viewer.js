// State variables
let ws = null;
let matchId = null;
let eventType = null;
let eventId = null;
// Optional token (viewer is allowed without auth)
let jwtToken = null;
let matchEnded = false;
let tossWinner = null;
let tossDecision = null;

function updateTossInfoUI(teamAName = 'Team A', teamBName = 'Team B') {
    const el = document.getElementById('toss-info');
    if (!el) return;
    if (!tossWinner) {
        el.textContent = 'Toss: —';
        return;
    }
    const winnerName = tossWinner === 'teamB' ? teamBName : teamAName;
    const decisionText = tossDecision === 'defend' ? 'Defend First' : 'Raid First';
    el.textContent = `Toss: ${winnerName} | Decided To: ${decisionText}`;
}

function normalizeEventType(value) {
    if (!value) return null;
    const type = String(value).toLowerCase();
    if (type === 'tournament' || type === 'championship') return type;
    return null;
}

function applyEventInfo(source) {
    if (!source) return;

    const candidateType = normalizeEventType(source.event_type || source.eventType);
    const candidateId = source.event_id || source.eventId;

    if (!eventType && candidateType) {
        eventType = candidateType;
    }
    if (!eventId && candidateId) {
        eventId = candidateId;
    }

    if (eventType && eventId) {
        updateEventNavButton();
    }
}

function renderScorecard(playerStats, teamAIds = [], teamBIds = [], teamAName = 'Team A', teamBName = 'Team B') {
    const listA = document.getElementById('scorecard-teamA-list');
    const listB = document.getElementById('scorecard-teamB-list');
    const labelA = document.getElementById('scorecard-teamA');
    const labelB = document.getElementById('scorecard-teamB');
    if (!listA || !listB || !labelA || !labelB) return;

    labelA.textContent = teamAName || 'Team A';
    labelB.textContent = teamBName || 'Team B';

    if (!playerStats || Object.keys(playerStats).length === 0) {
        listA.innerHTML = '<div style="color:#facc15;">No player stats yet.</div>';
        listB.innerHTML = '<div style="color:#facc15;">No player stats yet.</div>';
        return;
    }

    const entries = Object.entries(playerStats).map(([key, value]) => ({
        id: value.id || value.ID || key,
        data: value,
    }));

    const setA = new Set(teamAIds || []);
    const setB = new Set(teamBIds || []);

    const inA = [];
    const inB = [];
    const unknown = [];

    entries.forEach(p => {
        if (setA.has(p.id)) inA.push(p);
        else if (setB.has(p.id)) inB.push(p);
        else unknown.push(p);
    });

    if (setA.size === 0 && setB.size === 0) {
        const mid = 7;
        inA.push(...unknown.slice(0, mid));
        inB.push(...unknown.slice(mid, mid * 2));
    } else if (unknown.length) {
        inA.push(...unknown);
    }

    const renderCards = (arr) => {
        if (!arr.length) return '<div style="color:#facc15;">No players available.</div>';
        return arr.map(p => {
            const stat = p.data || {};
            const name = stat.name || stat.Name || 'Player';
            const raid = stat.raidPoints ?? stat.RaidPoints ?? 0;
            const def = stat.defencePoints ?? stat.DefencePoints ?? 0;
            const total = stat.totalPoints ?? stat.TotalPoints ?? 0;
            const status = (stat.status || stat.Status || '').toLowerCase();
            const statusBadge = status === 'out'
                ? '<span style="color:#f87171;font-weight:600;">OUT</span>'
                : '<span style="color:#34d399;font-weight:600;">IN</span>';
            const profileUrl = p.id ? `/playerprofile/${encodeURIComponent(p.id)}` : '#';

            return `
                <a href="${profileUrl}" style="text-decoration:none;color:inherit;">
                    <div style="border:1px solid rgba(255,255,255,0.1);background:rgba(30,41,59,0.7);border-radius:0.6rem;padding:0.75rem;margin-bottom:0.75rem;cursor:pointer;transition:all 0.2s ease;">
                        <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.35rem;">
                            <div style="font-weight:700;color:#fbbf24;">${name}</div>
                            <div>${statusBadge}</div>
                        </div>
                        <div style="display:flex;gap:1rem;flex-wrap:wrap;color:#e2e8f0;">
                            <div>Raid: <strong>${raid}</strong></div>
                            <div>Defence: <strong>${def}</strong></div>
                            <div>Total: <strong>${total}</strong></div>
                        </div>
                    </div>
                </a>
            `;
        }).join('');
    };

    listA.innerHTML = renderCards(inA);
    listB.innerHTML = renderCards(inB);
}

function joinMatch(id) {
    if (!id) return;
    
    matchId = id;
    // hide join UI
    const joinDiv = document.getElementById('viewer-join');
    if (joinDiv) joinDiv.style.display = 'none';
    const matchScoreBtn = document.getElementById('match-score-btn');
    if (matchScoreBtn) matchScoreBtn.disabled = false;

    // Try to fetch match metadata to get event info
    fetchMatchMetadata();

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
            
            // Match not initialized yet: show waiting message, don't mark ended
            if (data.error && data.error.toLowerCase().includes('not initialized')) {
                const infoEl = document.getElementById('viewer-info');
                if (infoEl) infoEl.textContent = 'Match not started yet. Please wait for the scorer to start the match.';
                const conn = document.getElementById('viewer-conn');
                if (conn) { conn.textContent = 'Waiting'; conn.style.background = '#6b7280'; }
                return;
            }

            // Check for error indicating match ended or not found
            if (data.error && data.error.toLowerCase().includes('ended')) {
                matchEnded = true;
                ws.close();
                showEndedMatchUI();
                return;
            }

            // Check for event information at top level (tournament/championship)
            applyEventInfo(data);
            
            if (data.type === "gameStats" || data.data) {
                const payload = data.data || data;
                applyEventInfo(payload);
                if (payload.teamA) document.getElementById("teamA-name").textContent = payload.teamA.name;
                if (payload.teamA) document.getElementById("teamA-score").textContent = payload.teamA.score;
                if (payload.teamB) document.getElementById("teamB-name").textContent = payload.teamB.name;
                if (payload.teamB) document.getElementById("teamB-score").textContent = payload.teamB.score;

                if (payload.tossWinner || payload.data?.tossWinner) tossWinner = payload.tossWinner || payload.data?.tossWinner;
                if (payload.tossDecision || payload.data?.tossDecision) tossDecision = payload.tossDecision || payload.data?.tossDecision;
                updateTossInfoUI(payload.teamA?.name || 'Team A', payload.teamB?.name || 'Team B');

                if (payload.playerStats) {
                    renderScorecard(
                        payload.playerStats,
                        payload.teamAPlayerIds || payload.teamAPlayerIDs || [],
                        payload.teamBPlayerIds || payload.teamBPlayerIDs || [],
                        payload.teamA?.name || 'Team A',
                        payload.teamB?.name || 'Team B'
                    );
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
    };
}

function showEndedMatchUI() {
    if (matchEnded) {
        // Ensure ended UI is visible and wired once
        const endedSection = document.getElementById('viewer-ended');
        if (endedSection) endedSection.style.display = 'block';
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

    // Try to fetch and show the final score from MongoDB
    tryFetchFinalScore();
}

function fetchMatchMetadata() {
    if (!matchId) return;
    
    // Try to get match details to check if it's part of tournament/championship
    fetch(`/api/match/${matchId}`)
        .then(res => {
            if (!res.ok) throw new Error('Match metadata not available');
            return res.json();
        })
        .then(match => {
            if (match && match.data) {
                const data = match.data;

                applyEventInfo(match);
                applyEventInfo(data);

                const inferredType = normalizeEventType(match.type || match.Type || data.type || data.Type);
                if (!eventType && inferredType) {
                    eventType = inferredType;
                }
                if (data.tossWinner || match.tossWinner) {
                    tossWinner = data.tossWinner || match.tossWinner;
                }
                if (data.tossDecision || match.tossDecision) {
                    tossDecision = data.tossDecision || match.tossDecision;
                }
                updateTossInfoUI(data.teamA?.name || 'Team A', data.teamB?.name || 'Team B');
                updateEventNavButton();
            }
        })
        .catch(err => {
            console.log('Could not fetch match metadata:', err);
        });
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

                if (data.playerStats) {
                    renderScorecard(
                        data.playerStats,
                        data.teamAPlayerIds || data.teamAPlayerIDs || [],
                        data.teamBPlayerIds || data.teamBPlayerIDs || [],
                        data.teamA?.name || 'Team A',
                        data.teamB?.name || 'Team B'
                    );
                }

                if (data.tossWinner || match.tossWinner) {
                    tossWinner = data.tossWinner || match.tossWinner;
                }
                if (data.tossDecision || match.tossDecision) {
                    tossDecision = data.tossDecision || match.tossDecision;
                }
                updateTossInfoUI(data.teamA?.name || 'Team A', data.teamB?.name || 'Team B');

                applyEventInfo(match);
                applyEventInfo(data);
            }
            if (!eventType) {
                const inferredType = normalizeEventType(match.type || match.Type);
                if (inferredType) eventType = inferredType;
            }
            updateEventNavButton();
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

    // Navigate to the public match score viewer page
    window.location.href = `/viewer/match/${matchId}`;
}

document.addEventListener('DOMContentLoaded', () => {
    const params = new URLSearchParams(window.location.search);
    const urlMatch = params.get('match_id');
    const urlEventType = params.get('event_type');
    const urlEventId = params.get('event_id');
    const urlToken = params.get('token');
    jwtToken = urlToken || null;
    eventType = urlEventType ? urlEventType.toLowerCase() : null;
    eventId = urlEventId || null;
    const joinInput = document.getElementById('viewer-match-id-input');
    const joinBtn = document.getElementById('viewer-join-btn');
    const info = document.getElementById('viewer-info');
    const matchScoreBtn = document.getElementById('match-score-btn');
    const typeSelect = document.getElementById('viewer-type-select');

    if (matchScoreBtn) {
        matchScoreBtn.disabled = true;
        matchScoreBtn.addEventListener('click', fetchMatchScore);
    }
    updateEventNavButton();

    const updatePlaceholder = () => {
        if (!joinInput || !typeSelect) return;
        const type = (typeSelect.value || 'match').toLowerCase();
        if (type === 'match') joinInput.placeholder = 'Enter match code to view';
        if (type === 'tournament') joinInput.placeholder = 'Enter tournament ID to view';
        if (type === 'championship') joinInput.placeholder = 'Enter championship ID to view';
    };
    if (typeSelect) {
        typeSelect.addEventListener('change', updatePlaceholder);
        updatePlaceholder();
    }

    if (urlMatch) {
        if (joinInput) joinInput.value = urlMatch;
        joinMatch(urlMatch);
    } else {
        // show join panel
        if (joinBtn) {
            joinBtn.addEventListener('click', () => {
                const v = joinInput ? joinInput.value.trim() : '';
                const selectedType = (typeSelect ? typeSelect.value : 'match').toLowerCase();
                if (!v) {
                    const msg = selectedType === 'match'
                        ? 'Enter a match code to join'
                        : selectedType === 'tournament'
                            ? 'Enter a tournament ID'
                            : 'Enter a championship ID';
                    return alert(msg);
                }
                if (selectedType === 'match') {
                    joinMatch(v);
                    return;
                }
                const target = `/viewer/${selectedType}/${encodeURIComponent(v)}`;
                window.location.href = target;
            });
        }
    }
});

function updateEventNavButton() {
    const eventBtn = document.getElementById('event-nav-btn');
    if (!eventBtn) return;
    const type = (eventType || '').toLowerCase();
    if ((type === 'tournament' || type === 'championship') && eventId) {
        const label = type === 'tournament' ? 'Go to Tournament' : 'Go to Championship';
        eventBtn.textContent = label;
        eventBtn.style.display = 'inline-block';
        eventBtn.onclick = () => {
            window.location.href = `/viewer/${type}/${encodeURIComponent(eventId)}`;
        };
    } else {
        eventBtn.style.display = 'none';
        eventBtn.onclick = null;
    }
}
