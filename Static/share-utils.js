(function () {
  let toastEl = null;
  let toastTimer = null;

  function ensureToastElement() {
    if (toastEl) return toastEl;
    toastEl = document.createElement('div');
    toastEl.id = 'raidx-share-toast';
    toastEl.style.position = 'fixed';
    toastEl.style.right = '16px';
    toastEl.style.bottom = '16px';
    toastEl.style.zIndex = '100000';
    toastEl.style.maxWidth = '360px';
    toastEl.style.padding = '10px 14px';
    toastEl.style.borderRadius = '8px';
    toastEl.style.fontSize = '14px';
    toastEl.style.fontWeight = '600';
    toastEl.style.color = '#fff';
    toastEl.style.boxShadow = '0 6px 20px rgba(0,0,0,0.35)';
    toastEl.style.opacity = '0';
    toastEl.style.transform = 'translateY(8px)';
    toastEl.style.transition = 'opacity 0.2s ease, transform 0.2s ease';
    toastEl.style.pointerEvents = 'none';
    toastEl.style.display = 'none';
    document.body.appendChild(toastEl);
    return toastEl;
  }

  function showToast(message, kind) {
    const el = ensureToastElement();
    const isError = String(kind || '').toLowerCase() === 'error';
    el.textContent = message || '';
    el.style.background = isError ? '#b91c1c' : '#166534';
    el.style.display = 'block';

    requestAnimationFrame(() => {
      el.style.opacity = '1';
      el.style.transform = 'translateY(0)';
    });

    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => {
      el.style.opacity = '0';
      el.style.transform = 'translateY(8px)';
      setTimeout(() => {
        el.style.display = 'none';
      }, 220);
    }, 1800);
  }

  function tryCopyText(text) {
    if (!text) return Promise.reject(new Error('No text to copy'));
    if (navigator.clipboard && navigator.clipboard.writeText) {
      return navigator.clipboard.writeText(text);
    }
    return new Promise((resolve, reject) => {
      try {
        const input = document.createElement('input');
        input.value = text;
        input.style.position = 'fixed';
        input.style.left = '-9999px';
        document.body.appendChild(input);
        input.select();
        const ok = document.execCommand('copy');
        document.body.removeChild(input);
        if (ok) resolve();
        else reject(new Error('copy failed'));
      } catch (err) {
        reject(err);
      }
    });
  }

  function buildViewerShareLink(type, id) {
    const safeId = id ? encodeURIComponent(String(id)) : '';
    switch (String(type || '').toLowerCase()) {
      case 'match':
        return `${location.origin}/viewer/match/${safeId}`;
      case 'match_overview':
        return `${location.origin}/viewer/match/${safeId}/overview`;
      case 'tournament':
        return `${location.origin}/viewer/tournament/${safeId}`;
      case 'championship':
        return `${location.origin}/viewer/championship/${safeId}`;
      case 'event_match':
        return `${location.origin}/viewer?event_type=match&event_id=${safeId}`;
      case 'live':
        return `${location.origin}/viewer?match_id=${safeId}`;
      default:
        return `${location.origin}/viewer`;
    }
  }

  async function shareViewerLink(url, options) {
    const opts = options || {};
    const title = opts.title || 'RaidX Viewer Link';
    const copiedMessage = opts.copiedMessage || 'Viewer link copied.';
    if (!url) {
      showToast('No link available to share.', 'error');
      return false;
    }

    if (navigator.share) {
      try {
        await navigator.share({
          title,
          text: url,
          url,
        });
        return true;
      } catch (_) {
      }
    }

    try {
      await tryCopyText(url);
      showToast(copiedMessage, 'success');
      return true;
    } catch (_) {
      showToast('Failed to copy link. Please copy manually from address bar.', 'error');
      return false;
    }
  }

  window.buildViewerShareLink = buildViewerShareLink;
  window.shareViewerLink = shareViewerLink;
})();
