document.addEventListener('DOMContentLoaded', () => {
    const role = (getRoleFromToken() || '').toLowerCase();
    const pageRole = (document.body.dataset.role || '').toLowerCase();

    const roleBadge = document.getElementById('role-badge');
    if (roleBadge) {
        roleBadge.textContent = role ? role.replace('_', ' ') : 'unknown';
    }

    if (pageRole && role && pageRole !== role) {
        const warn = document.getElementById('role-warning');
        if (warn) {
            warn.textContent = `You are logged in as ${role}. This page is for ${pageRole}.`;
            warn.classList.remove('d-none');
        }
    }

    if (pageRole === 'player') {
        initPlayerDashboard();
    }
    if (pageRole === 'team_owner') {
        initOwnerDashboard();
    }
    if (pageRole === 'organizer') {
        initOrganizerDashboard();
    }
});

function setStatus(id, text, type = 'info') {
    const el = document.getElementById(id);
    if (!el) return;
    el.className = `alert alert-${type}`;
    el.textContent = text;
    el.classList.remove('d-none');
}

function hideStatus(id) {
    const el = document.getElementById(id);
    if (!el) return;
    el.classList.add('d-none');
}

function formatId(id) {
    if (!id) return '';
    return typeof id === 'string' ? id : (id.$oid || id.Hex || id.hex || JSON.stringify(id));
}

function getInviteId(invite) {
    return invite?.ID || invite?._id || invite?.id || invite?.Id || '';
}

function getInviteField(invite, camelName, snakeName) {
    return invite?.[camelName] || invite?.[snakeName] || '';
}

async function updateInvite(inviteId, status, extra = {}) {
    try {
        const res = await apiRequest(`/api/invitations/${inviteId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ status, ...extra })
        });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error || 'Failed to update invite');

        if (document.getElementById('player-invites-pending')) {
            await loadPlayerInvites();
        }
        if (document.getElementById('owner-event-invites-pending') || document.getElementById('owner-event-invites-accepted') || document.getElementById('owner-event-invites-declined') || document.getElementById('owner-event-invites')) {
            await loadOwnerEventInvites();
        }
    } catch (e) {
        const statusEl = document.getElementById('player-status') || document.getElementById('owner-status');
        if (statusEl) {
            const statusId = statusEl.id;
            setStatus(statusId, e.message || 'Failed to update invite', 'danger');
        } else {
            console.error('Failed to update invite:', e);
        }
    }
}

async function initPlayerDashboard() {
    const profileLink = document.getElementById('player-profile-link');
    if (profileLink) {
        const userId = getUserIdFromToken();
        profileLink.href = userId ? `/playerprofile/${userId}` : '#';
    }
    const refreshBtn = document.getElementById('player-refresh');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadPlayerInvites);
    }
    const teamRefreshBtn = document.getElementById('player-teams-refresh');
    if (teamRefreshBtn) {
        teamRefreshBtn.addEventListener('click', async () => {
            await loadPlayerTeams();
            await loadPlayerEvents();
        });
    }
    await loadPlayerInvites();
    await loadPlayerTeams();
    await loadPlayerEvents();
}

async function loadPlayerInvites() {
    hideStatus('player-status');
    const pendingList = document.getElementById('player-invites-pending');
    const acceptedList = document.getElementById('player-invites-accepted');
    const declinedList = document.getElementById('player-invites-declined');
    if (!pendingList || !acceptedList || !declinedList) return;
    pendingList.innerHTML = '';
    acceptedList.innerHTML = '';
    declinedList.innerHTML = '';

    try {
        const res = await apiRequest('/api/invitations');
        const data = await res.json();
        if (!Array.isArray(data) || data.length === 0) {
            pendingList.innerHTML = '<div class="text-white">No pending invitations.</div>';
            acceptedList.innerHTML = '<div class="text-white">No accepted invitations.</div>';
            declinedList.innerHTML = '<div class="text-white">No declined invitations.</div>';
            return;
        }
        const pendingInvites = data.filter(inv => ['pending', 'invited_via_link'].includes((inv.status || inv.Status)));
        const acceptedInvites = data.filter(inv => (inv.status || inv.Status) === 'accepted' || (inv.status || inv.Status) === 'accepted_by_owner');
        const declinedInvites = data.filter(inv => (inv.status || inv.Status) === 'declined' || (inv.status || inv.Status) === 'declined_by_owner');

        renderPlayerInviteList(pendingList, pendingInvites, true);
        renderPlayerInviteList(acceptedList, acceptedInvites, false);
        renderPlayerInviteList(declinedList, declinedInvites, false);
    } catch (e) {
        setStatus('player-status', 'Failed to load invitations', 'danger');
    }
}

function renderPlayerInviteList(container, invites, showActions) {
    if (!container) return;
    if (!invites || invites.length === 0) {
        container.innerHTML = '<div class="text-white">No invitations.</div>';
        return;
    }

    invites.forEach(invite => {
        const card = document.createElement('div');
        card.className = 'request-card';
        const inviteId = invite?.id || getInviteId(invite);
        const statusValue = invite.status || invite.Status || 'pending';
        const statusDisplay = {
            'invited_via_link': 'waiting for owner approval',
            'accepted_by_owner': 'accepted by owner',
            'declined_by_owner': 'declined by owner',
            'pending': 'pending',
            'accepted': 'accepted',
            'declined': 'declined'
        }[statusValue] || statusValue;
        const canAct = showActions && statusValue !== 'invited_via_link';
        const declineReason = invite.declineReason || invite.decline_reason || invite.DeclineReason || '';
        card.innerHTML = `
            <h6>Team Invite</h6>
            <p class="mb-1">Team: ${invite.teamName || 'Unknown'}</p>
            <p class="mb-1">Owner: ${invite.ownerName || 'Unknown'}</p>
            <p class="mb-1">Invite ID: <span class="text-white">${formatId(inviteId)}</span></p>
            <p class="mb-1">Team ID: <span class="text-white">${formatId(invite.teamId || getInviteField(invite, 'TeamID', 'team_id'))}</span></p>
            <p>Status: <span class="badge-custom">${statusDisplay}</span></p>
            ${declineReason && statusValue.startsWith('declined') ? `<p class="mb-1">Reason: <span class="text-white">${declineReason}</span></p>` : ''}
            ${canAct ? `
            <div class="request-actions">
                <button class="btn btn-sm btn-success">Accept</button>
                <button class="btn btn-sm btn-outline-danger">Decline</button>
            </div>
            ` : ''}
        `;
        if (canAct) {
            const [acceptBtn, declineBtn] = card.querySelectorAll('button');
            acceptBtn.addEventListener('click', () => updateInvite(inviteId, 'accepted'));
            declineBtn.addEventListener('click', () => updateInvite(inviteId, 'declined'));
        }
        container.appendChild(card);
    });
}

async function loadPlayerTeams() {
    hideStatus('player-status');
    const list = document.getElementById('player-teams-list');
    if (!list) return;
    list.innerHTML = '';

    try {
        const res = await apiRequest('/api/player/teams');
        const data = await res.json();
        if (!Array.isArray(data) || data.length === 0) {
            list.innerHTML = '<div class="text-white">No teams found.</div>';
            return;
        }
        data.forEach(team => {
            const card = document.createElement('div');
            card.className = 'request-card';
            card.innerHTML = `
                <h6>${team.teamName || 'Team'}</h6>
                <p class="mb-1">Team ID: <span class="text-white">${formatId(team.teamId)}</span></p>
                ${team.description ? `<p class="mb-1">Description: ${team.description}</p>` : ''}
                <p>Status: <span class="badge-custom">${team.status || 'active'}</span></p>
            `;
            list.appendChild(card);
        });
    } catch (e) {
        setStatus('player-status', 'Failed to load teams', 'danger');
    }
}

async function loadPlayerEvents() {
    hideStatus('player-status');
    const list = document.getElementById('player-events-list');
    if (!list) return;
    list.innerHTML = '';

    try {
        const res = await apiRequest('/api/player/events');
        const data = await res.json();
        if (!Array.isArray(data) || data.length === 0) {
            list.innerHTML = '<div class="text-white">No events found.</div>';
            return;
        }
        data.forEach(evt => {
            const card = document.createElement('div');
            card.className = 'request-card';
            const typeLabel = evt.eventType ? evt.eventType.replace('_', ' ') : 'event';
            card.innerHTML = `
                <h6>${evt.eventName || 'Event'}</h6>
                <p class="mb-1">Type: <span class="badge-custom">${typeLabel}</span></p>
                ${evt.status ? `<p class="mb-1">Status: <span class="badge-custom">${evt.status}</span></p>` : ''}
                <p class="mb-1">Matches Played: <span class="text-white">${evt.matchCount || 0}</span></p>
                ${evt.eventId ? `<p class="mb-1">Event ID: <span class="text-white">${formatId(evt.eventId)}</span></p>` : ''}
            `;
            list.appendChild(card);
        });
    } catch (e) {
        setStatus('player-status', 'Failed to load events', 'danger');
    }
}

async function initOwnerDashboard() {
    const profileLink = document.getElementById('owner-profile-link');
    if (profileLink) {
        const userId = getUserIdFromToken();
        profileLink.href = userId ? `/owner/profile/${userId}` : '#';
    }
    const createForm = document.getElementById('owner-create-team');
    if (createForm) {
        createForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            hideStatus('owner-status');
            const teamName = document.getElementById('team-name').value.trim();
            const description = document.getElementById('team-desc').value.trim();
            try {
                const res = await apiRequest('/api/teams', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ team_name: teamName, description })
                });
                const data = await res.json();
                if (!res.ok) throw new Error(data.error || 'Failed to create team');
                const newTeamId = formatId(data.team_id);
                if (teamIdInput) teamIdInput.value = newTeamId;
                localStorage.setItem('rbac_team_id', newTeamId);
                setStatus('owner-status', `Team created: ${data.team_name} (${newTeamId})`, 'success');
            } catch (e) {
                setStatus('owner-status', e.message, 'danger');
            }
        });
    }

    const inviteForm = document.getElementById('owner-invite-player');
    if (inviteForm) {
        inviteForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            hideStatus('owner-status');
            const teamId = (teamIdInput && teamIdInput.value.trim()) || '';
            const playerId = document.getElementById('player-id').value.trim();
            const username = document.getElementById('player-username').value.trim();
            const generateLink = document.getElementById('generate-link').checked;
            try {
                const res = await apiRequest(`/api/teams/${teamId}/invite`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ player_id: playerId, username, generate_link: generateLink })
                });
                const data = await res.json();
                if (!res.ok) throw new Error(data.error || 'Failed to invite player');
                const linkBox = document.getElementById('invite-link');
                if (linkBox) linkBox.value = data.invite_url || '';
                setStatus('owner-status', 'Invitation created.', 'success');
            } catch (e) {
                setStatus('owner-status', e.message, 'danger');
            }
        });
    }

    const refreshInvitesBtn = document.getElementById('owner-refresh');
    if (refreshInvitesBtn) refreshInvitesBtn.addEventListener('click', loadOwnerTeamInvites);
    await loadOwnerTeamInvites();

    const refreshEventInvitesBtn = document.getElementById('owner-event-refresh');
    if (refreshEventInvitesBtn) refreshEventInvitesBtn.addEventListener('click', loadOwnerEventInvites);

    const ownerEventTabBtn = document.getElementById('tournaments-tab');
    if (ownerEventTabBtn) {
        ownerEventTabBtn.addEventListener('shown.bs.tab', () => {
            loadOwnerEventInvites();
        });
    }

    await loadOwnerEventInvites();
}

async function loadOwnerTeamInvites() {
    const list = document.getElementById('owner-invites');
    const teamId = document.getElementById('owner-team-id')?.value.trim();
    if (!list || !teamId) return;
    list.innerHTML = '';

    try {
        const res = await apiRequest(`/api/teams/${teamId}/invites`);
        const data = await res.json();
        if (!Array.isArray(data) || data.length === 0) {
            list.innerHTML = '<div class="text-white">No invites found for this team.</div>';
            return;
        }
        data.forEach(invite => {
            const card = document.createElement('div');
            card.className = 'request-card';
            const inviteId = getInviteId(invite);
            const statusValue = (getInviteField(invite, 'Status', 'status') || '').toLowerCase();
            const statusDisplay = {
                'invited_via_link': 'waiting for player action',
                'pending': 'pending',
                'accepted': 'accepted',
                'declined': 'declined'
            }[statusValue] || (getInviteField(invite, 'Status', 'status') || 'pending');
            card.innerHTML = `
                <h6>Player Invite</h6>
                <p class="mb-1">Invite ID: <span class="text-white">${formatId(inviteId)}</span></p>
                <p class="mb-1">Player ID: <span class="text-white">${formatId(getInviteField(invite, 'ToID', 'to_id'))}</span></p>
                <p>Status: <span class="badge-custom">${statusDisplay}</span></p>
            `;
            list.appendChild(card);
        });
    } catch (e) {
        setStatus('owner-status', 'Failed to load team invites', 'danger');
    }
}

async function loadOwnerEventInvites() {
    const pendingList = document.getElementById('owner-event-invites-pending');
    const acceptedList = document.getElementById('owner-event-invites-accepted');
    const declinedList = document.getElementById('owner-event-invites-declined');
    const legacyList = document.getElementById('owner-event-invites');
    if (!pendingList && !acceptedList && !declinedList && !legacyList) return;
    if (pendingList) pendingList.innerHTML = '';
    if (acceptedList) acceptedList.innerHTML = '';
    if (declinedList) declinedList.innerHTML = '';
    if (legacyList) legacyList.innerHTML = '';

    const setVisibleMessage = (message) => {
        if (pendingList) pendingList.innerHTML = `<div class="text-white">${message}</div>`;
        if (acceptedList) acceptedList.innerHTML = `<div class="text-white">${message}</div>`;
        if (declinedList) declinedList.innerHTML = `<div class="text-white">${message}</div>`;
        if (legacyList) legacyList.innerHTML = `<div class="text-white">${message}</div>`;
    };

    let ownerTeams = [];
    try {
        const teamsRes = await apiRequest('/api/owner/teams');
        const teamsJson = await teamsRes.json();
        ownerTeams = Array.isArray(teamsJson) ? teamsJson : (teamsJson.data || []);
    } catch (e) {
        ownerTeams = [];
    }

    try {
        // Load ALL event invitations (not just pending)
        const res = await apiRequest('/api/owner/event-invitations');
        const data = await res.json();
        if (!res.ok) {
            throw new Error(data.error || 'Failed to load event invites');
        }
        if (!Array.isArray(data) || data.length === 0) {
            setVisibleMessage('No event invites.');
            return;
        }
        
        // Separate by status
        const getStatus = (inv) => (inv.status || inv.Status || '').toLowerCase();
        const pending = data.filter(inv => ['pending', 'invited_via_link'].includes(getStatus(inv)));
        const accepted = data.filter(inv => ['accepted', 'accepted_by_organizer', 'accepted_by_owner'].includes(getStatus(inv)));
        const declined = data.filter(inv => ['declined', 'declined_by_organizer'].includes(getStatus(inv)));
        
        // Render into tab containers when present, otherwise fallback to legacy list
        if (pendingList && acceptedList && declinedList) {
            if (pending.length === 0) pendingList.innerHTML = '<div class="text-white">No pending invites.</div>';
            if (accepted.length === 0) acceptedList.innerHTML = '<div class="text-white">No accepted invites.</div>';
            if (declined.length === 0) declinedList.innerHTML = '<div class="text-white">No declined invites.</div>';
            pending.forEach(invite => renderOwnerEventInvite(pendingList, invite, ownerTeams, true));
            accepted.forEach(invite => renderOwnerEventInvite(acceptedList, invite, ownerTeams, false));
            declined.forEach(invite => renderOwnerEventInvite(declinedList, invite, ownerTeams, false));
        } else if (legacyList) {
            pending.forEach(invite => renderOwnerEventInvite(legacyList, invite, ownerTeams, true));
            accepted.forEach(invite => renderOwnerEventInvite(legacyList, invite, ownerTeams, false));
            declined.forEach(invite => renderOwnerEventInvite(legacyList, invite, ownerTeams, false));
            if (pending.length === 0 && accepted.length === 0 && declined.length === 0) {
                legacyList.innerHTML = '<div class="text-white">No event invites.</div>';
            }
        }
    } catch (e) {
        setVisibleMessage('Failed to load event invites.');
        setStatus('owner-status', 'Failed to load event invites', 'danger');
    }
}

function renderOwnerEventInvite(list, invite, ownerTeams, showActions) {
    const inviteId = getInviteId(invite);
    console.log('DEBUG renderOwnerEventInvite - Processing invite, ID:', inviteId, 'Full invite:', invite);
    const card = document.createElement('div');
    card.className = 'request-card';
    card.setAttribute('data-invite-id', inviteId); // Add for debugging
    const rawStatus = (invite.status || invite.Status || '').toLowerCase();
    const statusBadge = ['pending', 'invited_via_link'].includes(rawStatus) ? 'badge bg-warning' : 
                       ['accepted', 'accepted_by_organizer'].includes(rawStatus) ? 'badge bg-success' : 'badge bg-danger';
    const statusDisplay = {
        'invited_via_link': 'waiting for organizer approval',
        'accepted_by_organizer': 'accepted by organizer',
        'declined_by_organizer': 'declined by organizer',
        'pending': 'pending',
        'accepted': 'accepted',
        'declined': 'declined'
    }[rawStatus] || rawStatus;
    const declineReason = invite.declineReason || invite.decline_reason || invite.DeclineReason || '';
    const eventName = invite.eventName || invite.EventName || 'Unknown Event';
    const eventType = invite.eventType || invite.EventType || 'Unknown Type';
    const ownerName = invite.ownerName || invite.OwnerName || 'Unknown Owner';
    const teamName = invite.teamName || invite.TeamName || 'Unassigned';
    const teamOptions = Array.isArray(ownerTeams) ? ownerTeams : (ownerTeams.data || []);
    const canAct = showActions && rawStatus !== 'invited_via_link';
    card.innerHTML = `
        <h6>Event Invite</h6>
        <p class="mb-1">Team Owner: ${ownerName}</p>
        <p class="mb-1">Event: <span class="text-white">${eventName}</span></p>
        <p class="mb-1">Event Type: <span class="text-white">${eventType}</span></p>
        <p class="mb-1">Team: <span class="text-white">${teamName}</span></p>
        <p>Status: <span class="badge-custom">${statusDisplay}</span></p>
        ${declineReason && rawStatus.startsWith('declined') ? `<p class="mb-1">Reason: <span class="text-white">${declineReason}</span></p>` : ''}
        ${canAct ? `
        <div class="request-actions">
            <select class="form-select form-select-sm" style="max-width: 220px;">
                <option value="">Select team to accept...</option>
                ${teamOptions.map(t => {
                    const playersCount = Number(t.Players || 0);
                    const disabled = playersCount < 7 ? 'disabled' : '';
                    const label = `${t.TeamName} (${playersCount} players)`;
                    return `<option value="${t.ID}" ${disabled}>${label}</option>`;
                }).join('')}
            </select>
            <button class="btn btn-sm btn-success" data-action="accept">Accept</button>
            <button class="btn btn-sm btn-outline-danger" data-action="decline">Decline</button>
        </div>` : ''}
    `;
    if (canAct) {
        const selectEl = card.querySelector('select');
        const acceptBtn = card.querySelector('button[data-action="accept"]');
        const declineBtn = card.querySelector('button[data-action="decline"]');
        if (!acceptBtn || !declineBtn || !selectEl) {
            list.appendChild(card);
            return;
        }
        acceptBtn.addEventListener('click', () => {
            console.log('DEBUG Accept clicked for inviteId:', inviteId);
            const selectedTeamId = selectEl ? selectEl.value.trim() : '';
            if (!selectedTeamId) {
                setStatus('owner-status', 'Select a team before accepting.', 'warning');
                return;
            }
            const selectedTeam = teamOptions.find(t => t.ID === selectedTeamId);
            const playersCount = selectedTeam ? Number(selectedTeam.Players || 0) : 0;
            if (playersCount < 7) {
                setStatus('owner-status', 'Selected team must have at least 7 players.', 'warning');
                return;
            }
            console.log('DEBUG Calling updateInvite with inviteId:', inviteId, 'team_id:', selectedTeamId);
            updateInvite(inviteId, 'accepted', { team_id: selectedTeamId });
        });
        declineBtn.addEventListener('click', () => {
            console.log('DEBUG Decline clicked for inviteId:', inviteId);
            updateInvite(inviteId, 'declined');
        });
    }
    list.appendChild(card);
}

async function initOrganizerDashboard() {
    const profileLink = document.getElementById('organizer-profile-link');
    if (profileLink) {
        const userId = getUserIdFromToken();
        profileLink.href = userId ? `/organizer/profile/${userId}` : '#';
    }

    const eventTypeSelect = document.getElementById('event-type');
    const maxTeamsGroup = document.getElementById('max-teams-group');
    const maxTeamsInput = document.getElementById('event-max-teams');
    if (eventTypeSelect && maxTeamsGroup) {
        const toggleMaxTeams = () => {
            const type = eventTypeSelect.value;
            const shouldShow = (type === 'tournament' || type === 'championship');
            maxTeamsGroup.style.display = shouldShow ? 'block' : 'none';
            if (maxTeamsInput) {
                maxTeamsInput.required = shouldShow;
            }
        };
        eventTypeSelect.addEventListener('change', toggleMaxTeams);
        toggleMaxTeams();
    }

    const eventIdInput = document.getElementById('organizer-event-id');
    const storedEventId = localStorage.getItem('rbac_event_id');
    if (eventIdInput && storedEventId) eventIdInput.value = storedEventId;

    const createForm = document.getElementById('organizer-create-event');
    if (createForm) {
        createForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            hideStatus('organizer-status');
            const name = document.getElementById('event-name').value.trim();
            const type = document.getElementById('event-type').value;
            const maxTeams = parseInt(document.getElementById('event-max-teams')?.value || '0', 10);
            if ((type === 'tournament' || type === 'championship') && maxTeams <= 0) {
                setStatus('organizer-status', 'Please enter number of teams for tournaments/championships.', 'warning');
                return;
            }
            try {
                const res = await apiRequest('/api/events', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ event_name: name, event_type: type, max_teams: maxTeams })
                });
                const data = await res.json();
                if (!res.ok) throw new Error(data.error || 'Failed to create event');
                const newEventId = formatId(data.event_id);
                if (eventIdInput) eventIdInput.value = newEventId;
                localStorage.setItem('rbac_event_id', newEventId);
                setStatus('organizer-status', `Event created: ${data.event_name} (${newEventId})`, 'success');
                if (newEventId) {
                    window.location.href = `/organizer/event/${newEventId}`;
                    return;
                }
                await loadOrganizerEvents();
            } catch (e) {
                setStatus('organizer-status', e.message, 'danger');
            }
        });
    }
    const refreshEventsBtn = document.getElementById('organizer-refresh-events');
    if (refreshEventsBtn) refreshEventsBtn.addEventListener('click', loadOrganizerEvents);
    await loadOrganizerEvents();

}

async function loadOrganizerEvents() {
    const ongoingList = document.getElementById('organizer-events-ongoing');
    const pendingList = document.getElementById('organizer-events-pending');
    const completedList = document.getElementById('organizer-events-completed');
    if (!ongoingList || !pendingList || !completedList) {
        console.error('DEBUG: Missing organizer event containers');
        return;
    }
    ongoingList.innerHTML = '';
    pendingList.innerHTML = '';
    completedList.innerHTML = '';

    try {
        const res = await apiRequest('/api/organizer/events');
        const data = await res.json();
        console.log('DEBUG loadOrganizerEvents - Response status:', res.status);
        console.log('DEBUG loadOrganizerEvents - Data:', data);
        if (!res.ok) {
            throw new Error(data.error || 'Failed to load events');
        }
        if (!Array.isArray(data) || data.length === 0) {
            console.log('DEBUG: No events found');
            ongoingList.innerHTML = '<div class="text-white">No ongoing events.</div>';
            pendingList.innerHTML = '<div class="text-white">No pending events.</div>';
            completedList.innerHTML = '<div class="text-white">No completed events.</div>';
            return;
        }

        console.log('DEBUG: Found', data.length, 'total events');
        const completedEvents = data.filter(evt => (evt.status || '').toLowerCase() === 'completed');
        const ongoingEvents = data.filter(evt => {
            const status = (evt.status || '').toLowerCase();
            return status === 'ongoing' || status === 'active';
        });
        const pendingEvents = data.filter(evt => {
            const status = (evt.status || '').toLowerCase();
            return status !== 'completed' && status !== 'ongoing' && status !== 'active';
        });
        console.log('DEBUG: Ongoing:', ongoingEvents.length, 'Pending:', pendingEvents.length, 'Completed:', completedEvents.length);

        renderOrganizerEventList(ongoingList, ongoingEvents, true);
        renderOrganizerEventList(pendingList, pendingEvents, true);
        renderOrganizerEventList(completedList, completedEvents, false);
    } catch (e) {
        console.error('DEBUG loadOrganizerEvents error:', e);
        setStatus('organizer-status', e.message || 'Failed to load events', 'danger');
    }
}

function renderOrganizerEventList(container, events, allowActions) {
    if (!events || events.length === 0) {
        container.innerHTML = '<div class="text-white">No events available.</div>';
        return;
    }

    events.forEach(event => {
        const card = document.createElement('div');
        card.className = 'event-card';
        card.style.cursor = 'pointer';
        card.innerHTML = `
            <h5>${event.eventName}</h5>
            <p><strong>Type:</strong> ${event.eventType}</p>
            ${event.maxTeams ? `<p><strong>Max Teams:</strong> ${event.maxTeams}</p>` : ''}
            <p><strong>Status:</strong> <span class="badge-custom">${event.status === 'completed' ? 'completed' : event.status}</span></p>
            <p><strong>Accepted:</strong> ${event.counts?.accepted || 0} | <strong>Pending:</strong> ${event.counts?.pending || 0} | <strong>Declined:</strong> ${event.counts?.declined || 0}</p>
            ${allowActions ? `` : ''}
        `;
        card.addEventListener('click', (e) => {
            if (e.target.closest('button, input, select, textarea, label, a, .event-card-actions')) {
                return;
            }
            window.location.href = `/organizer/event/${event.id}`;
        });
        container.appendChild(card);
    });
}

function populateEventEdit(eventId) {
    const editSection = document.getElementById(`event-edit-${eventId}`);
    if (editSection) editSection.style.display = editSection.style.display === 'none' ? 'block' : 'none';
}

function toggleInviteForm(eventId) {
    const form = document.getElementById(`event-invite-${eventId}`);
    if (form) form.style.display = form.style.display === 'none' ? 'block' : 'none';
}

async function saveEventEdit(eventId) {
    const name = document.getElementById(`edit-name-${eventId}`)?.value.trim();
    const type = document.getElementById(`edit-type-${eventId}`)?.value;
    const maxTeams = parseInt(document.getElementById(`edit-max-${eventId}`)?.value || '0', 10);
    if (!name) return;
    if ((type === 'tournament' || type === 'championship') && maxTeams <= 0) {
        setStatus('organizer-status', 'Please enter number of teams for tournaments/championships.', 'warning');
        return;
    }

    try {
        const res = await apiRequest(`/api/events/${eventId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ event_name: name, event_type: type, max_teams: maxTeams })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'Failed to update event');
        }
        await loadOrganizerEvents();
    } catch (e) {
        setStatus('organizer-status', e.message, 'danger');
    }
}

async function sendOrganizerInvite(eventId) {
    const ownerInput = document.getElementById(`invite-owner-${eventId}`);
    const ownerIdentifier = ownerInput ? ownerInput.value.trim() : '';
    if (!ownerIdentifier) {
        setStatus('organizer-status', 'Enter team owner username or email.', 'warning');
        return;
    }

    try {
        const res = await apiRequest(`/api/events/${eventId}/invite`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ownerIdentifier })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed to invite team owner');
        setStatus('organizer-status', 'Invitation created.', 'success');
        if (ownerInput) ownerInput.value = '';
    } catch (e) {
        setStatus('organizer-status', e.message, 'danger');
    }
}

async function markEventDone(eventId) {
    try {
        const res = await apiRequest(`/api/events/${eventId}/complete`, {
            method: 'POST'
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed to update event');
        await loadOrganizerEvents();
    } catch (e) {
        setStatus('organizer-status', e.message, 'danger');
    }
}

async function loadOrganizerRequests() {
    const pendingList = document.getElementById('organizer-requests-pending');
    const acceptedList = document.getElementById('organizer-requests-accepted');
    const declinedList = document.getElementById('organizer-requests-declined');
    if (!pendingList || !acceptedList || !declinedList) {
        console.error('DEBUG: Missing organizer request containers');
        return;
    }
    pendingList.innerHTML = '';
    acceptedList.innerHTML = '';
    declinedList.innerHTML = '';

    try {
        const res = await apiRequest('/api/organizer/event-invites');
        const data = await res.json();
        console.log('DEBUG loadOrganizerRequests - Response status:', res.status);
        console.log('DEBUG loadOrganizerRequests - Data:', data);
        if (!res.ok) {
            throw new Error(data.error || 'Failed to load requests');
        }
        if (!Array.isArray(data) || data.length === 0) {
            console.log('DEBUG: No requests found');
            pendingList.innerHTML = '<div class="text-white">No pending requests.</div>';
            acceptedList.innerHTML = '<div class="text-white">No accepted requests.</div>';
            declinedList.innerHTML = '<div class="text-white">No declined requests.</div>';
            return;
        }

        console.log('DEBUG: Found', data.length, 'total requests');
        const pending = data.filter(inv => inv.status === 'pending');
        const accepted = data.filter(inv => inv.status === 'accepted');
        const declined = data.filter(inv => inv.status === 'declined');
        console.log('DEBUG: Pending:', pending.length, 'Accepted:', accepted.length, 'Declined:', declined.length);

        renderOrganizerRequestList(pendingList, pending);
        renderOrganizerRequestList(acceptedList, accepted);
        renderOrganizerRequestList(declinedList, declined);
    } catch (e) {
        console.error('DEBUG loadOrganizerRequests error:', e);
        setStatus('organizer-status', e.message || 'Failed to load requests', 'danger');
    }
}

function renderOrganizerRequestList(container, requests) {
    if (!requests || requests.length === 0) {
        container.innerHTML = '<div style="color: #fbbf24;">No requests.</div>';
        return;
    }
    requests.forEach(req => {
        const card = document.createElement('div');
        card.className = 'request-card';
        const declineReason = req.declineReason || req.decline_reason || req.DeclineReason || '';
        const teamName = req.teamName || 'Unassigned';
        const teamId = req.teamId || 'Unassigned';
        card.innerHTML = `
            <h6>${req.eventName || 'Event'}</h6>
            <p class="mb-1">Team Owner: ${req.ownerName || 'Unknown'}</p>
            <p class="mb-1">User ID: ${req.ownerUserId || 'N/A'}</p>
            <p class="mb-1">Team: <span class="text-white">${teamName}</span></p>
            <p class="mb-1">Team ID: <span class="text-white">${teamId}</span></p>
            <p>Status: <span class="badge-custom">${req.status}</span></p>
            ${declineReason && (req.status || '').startsWith('declined') ? `<p class="mb-1">Reason: <span class="text-white">${declineReason}</span></p>` : ''}
            <p class="text-muted mb-0">Invite ID: ${req.id}</p>
        `;
        container.appendChild(card);
    });
}

async function loadOrganizerEventTeams() {
    const list = document.getElementById('organizer-event-teams');
    const eventId = document.getElementById('organizer-event-id')?.value.trim();
    if (!list || !eventId) return;
    list.innerHTML = '';

    try {
        const res = await apiRequest(`/api/events/${eventId}/teams`);
        const data = await res.json();
        if (!Array.isArray(data) || data.length === 0) {
            list.innerHTML = '<div class="text-muted">No teams linked to this event.</div>';
            return;
        }
        data.forEach(entry => {
            const card = document.createElement('div');
            card.className = 'card mb-2';
            card.innerHTML = `
                <div class="card-body">
                    <div class="fw-semibold">Team: ${formatId(entry.team_id)}</div>
                    <div class="small text-muted">Status: ${entry.status}</div>
                </div>
            `;
            list.appendChild(card);
        });
    } catch (e) {
        setStatus('organizer-status', 'Failed to load event teams', 'danger');
    }
}
