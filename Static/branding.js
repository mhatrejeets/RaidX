(function () {
  const LOGO_PATH = '/static/raidxlogo-removebg-preview.png';
  const NAV_LOGO_SIZE = 80;

  function ensureFavicon() {
    let favicon = document.querySelector('link[rel="icon"], link[rel="shortcut icon"]');
    if (!favicon) {
      favicon = document.createElement('link');
      favicon.setAttribute('rel', 'icon');
      document.head.appendChild(favicon);
    }
    favicon.setAttribute('type', 'image/png');
    favicon.setAttribute('href', LOGO_PATH);
  }

  function ensureBrandStyles() {
    if (document.getElementById('raidx-branding-style')) return;

    const style = document.createElement('style');
    style.id = 'raidx-branding-style';
    style.textContent = `
      .navbar-brand.raidx-branded,
      span.navbar-brand.raidx-branded {
        display: inline-flex;
        align-items: center;
        gap: 8px;
      }

      .raidx-nav-logo {
        width: ${NAV_LOGO_SIZE}px;
        height: ${NAV_LOGO_SIZE}px;
        object-fit: contain;
        flex-shrink: 0;
      }
    `;
    document.head.appendChild(style);
  }

  function applyNavbarLogos() {
    const brandNodes = document.querySelectorAll('.navbar-brand');
    brandNodes.forEach((brandNode) => {
      if (brandNode.querySelector('.raidx-nav-logo')) return;

      const brandText = (brandNode.textContent || '').trim().toLowerCase();
      if (!brandText.includes('raidx')) return;

      const logoImage = document.createElement('img');
      logoImage.src = LOGO_PATH;
      logoImage.alt = 'RaidX';
      logoImage.className = 'raidx-nav-logo';

      brandNode.classList.add('raidx-branded');
      brandNode.prepend(logoImage);
    });
  }

  function applyBranding() {
    if (!document.head) return;
    ensureBrandStyles();
    ensureFavicon();
    applyNavbarLogos();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', applyBranding);
  } else {
    applyBranding();
  }
})();
