/**
 * Dark Mode Toggle Module
 * Handles theme switching between light and dark modes
 */
const DarkModeToggle = (() => {
  const STORAGE_KEY = 'theme-mode';
  const THEME_DARK = 'dark';
  const THEME_LIGHT = 'light';

  // Get system preference
  const getSystemTheme = () => {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? THEME_DARK : THEME_LIGHT;
  };

  // Get stored theme or system preference
  const getTheme = () => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) return stored;
    return getSystemTheme();
  };

  const updateToggleIcons = (theme) => {
    const icons = document.querySelectorAll('.theme-toggle-btn i');
    icons.forEach(icon => {
      if (theme === THEME_DARK) {
        icon.className = 'fas fa-sun';
      } else {
        icon.className = 'fas fa-moon';
      }
    });
  };

  // Apply theme to document
  const applyTheme = (theme) => {
    const html = document.documentElement;
    if (theme === THEME_DARK) {
      html.classList.add('dark');
    } else {
      html.classList.remove('dark');
    }
    localStorage.setItem(STORAGE_KEY, theme);
    updateToggleIcons(theme);
  };

  // Initialize theme
  const init = () => {
    const theme = getTheme();
    applyTheme(theme);
    
    // Ensure icons are updated once the DOM is loaded
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => updateToggleIcons(theme));
    } else {
      updateToggleIcons(theme);
    }
  };

  // Toggle theme
  const toggle = () => {
    const current = getTheme();
    const newTheme = current === THEME_DARK ? THEME_LIGHT : THEME_DARK;
    applyTheme(newTheme);
    window.dispatchEvent(new CustomEvent('themechanged', { detail: { theme: newTheme } }));
    return newTheme;
  };

  // Initialize on load
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  return {
    toggle,
    getTheme,
    applyTheme,
    init
  };
})();
