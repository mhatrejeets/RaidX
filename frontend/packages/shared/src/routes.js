export const ROUTE_TEMPLATES = {
  playerDashboard: "/player/dashboard",
  publicRankings: "/api/public/rankings/:type/:id",
  pendingApprovalApprove: "/api/pending-approvals/:id/approve",
  pendingApprovalReject: "/api/pending-approvals/:id/reject",
  teamInviteDetails: "/api/invite-link/team/:token/details",
  teamInviteAccept: "/api/invite-link/team/:token/accept",
  teamInviteClaim: "/api/invite-link/team/:token/claim",
  eventInviteDetails: "/api/invite-link/event/:token/details",
  eventInviteAccept: "/api/invite-link/event/:token/accept",
  eventInviteClaim: "/api/invite-link/event/:token/claim"
};

export function fillRoute(template, params) {
  let result = template;
  Object.entries(params || {}).forEach(([key, value]) => {
    result = result.replace(`:${key}`, String(value));
  });
  return result;
}
