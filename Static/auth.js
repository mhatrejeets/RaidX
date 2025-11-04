// Constants
const JWT_STORAGE_KEY = 'token';

// Get a valid token from localStorage
function getValidToken() {
    const token = localStorage.getItem(JWT_STORAGE_KEY);
    if (!token) return null;
    return token;
}

// Function to check if user is authenticated
function isAuthenticated() {
    return !!getValidToken();
}

// Require authentication or redirect to login
function requireAuth() {
    if (!isAuthenticated()) {
        const currentUrl = encodeURIComponent(window.location.href);
        window.location.href = `/login?returnUrl=${currentUrl}`;
        return false;
    }
    return true;
}

// Helper function to make authenticated API requests
async function apiRequest(url, options = {}) {
    const token = getValidToken();
    if (!token) {
        const currentUrl = encodeURIComponent(window.location.href);
        window.location.href = `/login?returnUrl=${currentUrl}`;
        throw new Error('No JWT token found');
    }

    // Add auth header to request
    const headers = {
        'Authorization': `Bearer ${token}`,
        ...options.headers,
    };

    try {
        const response = await fetch(url, {
            ...options,
            headers
        });

        if (response.status === 401) {
            localStorage.removeItem(JWT_STORAGE_KEY);
            const currentUrl = encodeURIComponent(window.location.href);
            window.location.href = `/login?returnUrl=${currentUrl}`;
            throw new Error('Unauthorized');
        }

        return response;
    } catch (error) {
        console.error('API request failed:', error);
        throw error;
    }
}

// Add authentication to all fetch requests
function setupGlobalAuth() {
    const originalFetch = window.fetch;
    window.fetch = function(url, options = {}) {
        const token = getValidToken();
        if (token) {
            options.headers = options.headers || {};
            options.headers['Authorization'] = `Bearer ${token}`;
        }
        return originalFetch(url, options);
    };
}

// Get authenticated URL (adds token as query parameter)
function getAuthenticatedUrl(url) {
    const token = getValidToken();
    if (!token) return url;
    const separator = url.includes('?') ? '&' : '?';
    return `${url}${separator}token=${token}`;
}

// Extract user id from JWT token (client-side). Returns null if not found.
function getUserIdFromToken() {
    const token = getValidToken();
    if (!token) return null;
    try {
        const parts = token.split('.');
        if (parts.length < 2) return null;
        // base64url decode payload
        const b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
        const json = decodeURIComponent(atob(b64).split('').map(function(c) {
            return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
        }).join(''));
        const payload = JSON.parse(json);
        return payload.user_id || payload.userId || payload.sub || payload.session_id || payload.sessionId || null;
    } catch (e) {
        console.warn('Failed to parse token for user id', e);
        return null;
    }
}

// Setup authenticated link
function setupAuthenticatedLink(elementId, basePath) {
    const element = document.getElementById(elementId);
    if (element) {
        element.href = getAuthenticatedUrl(basePath);
        element.addEventListener('click', (e) => {
            if (!isAuthenticated()) {
                e.preventDefault();
                window.location.href = '/login';
            }
        });
    }
}

// Function to handle logout
async function logout() {
    try {
        await apiRequest('/logout', { method: 'POST' });
    } catch (error) {
        console.error('Logout failed:', error);
    } finally {
        localStorage.removeItem(JWT_STORAGE_KEY);
        window.location.href = '/login';
    }
}