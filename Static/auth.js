// Constants (guard against duplicate script includes)
var JWT_STORAGE_KEY = window.JWT_STORAGE_KEY || 'token';
window.JWT_STORAGE_KEY = JWT_STORAGE_KEY;

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
    
    // Simple approach: remove any existing token parameter and add new one
    // Remove old token params using regex
    const cleanUrl = url.replace(/[?&]token=[^&]*/g, '');
    const separator = cleanUrl.includes('?') ? '&' : '?';
    const result = `${cleanUrl}${separator}token=${token}`;


    return result;
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

// Extract role from JWT token (client-side). Returns null if not found.
function getRoleFromToken() {
    const token = getValidToken();
    if (!token) return null;
    try {
        const parts = token.split('.');
        if (parts.length < 2) return null;
        const b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
        const json = decodeURIComponent(atob(b64).split('').map(function(c) {
            return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
        }).join(''));
        const payload = JSON.parse(json);
        return (payload.role || payload.Role || null);
    } catch (e) {
        console.warn('Failed to parse token for role', e);
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
        const response = await fetch('/logout', { 
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        // Always clear local storage regardless of response
        localStorage.clear();
        
        if (response.ok) {
            window.location.replace('/login');
        } else {
            console.error('Logout failed:', response.statusText);
            window.location.replace('/login');
        }
    } catch (error) {
        console.error('Logout error:', error);
        localStorage.clear();
        window.location.replace('/login');
    }
}

// Function to refresh access token using refresh token
async function refreshAccessToken() {
    try {
        const response = await fetch('/refresh', { 
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        
        if (response.ok) {
            const data = await response.json();
            localStorage.setItem(JWT_STORAGE_KEY, data.token);
            return true;
        } else if (response.status === 401) {
            // Refresh token expired or invalid
            localStorage.removeItem(JWT_STORAGE_KEY);
            window.location.href = '/login';
            return false;
        }
        return false;
    } catch (error) {
        console.error('Token refresh failed:', error);
        return false;
    }
}