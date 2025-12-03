// ui.js - accessibility & theming enhancements
// Provides: dark mode toggle persisted in localStorage, focus ring management, mobile nav menu.
// Dependencies: none (vanilla JS). Safe to import early.

const THEME_KEY = 'gauth-theme';
let currentTheme = 'light';

function applyTheme(theme) {
	currentTheme = theme === 'dark' ? 'dark' : 'light';
	const root = document.documentElement;
	if (currentTheme === 'dark') {
		root.classList.add('dark');
	} else {
		root.classList.remove('dark');
	}
	const toggle = document.getElementById('themeToggle');
	if (toggle) {
		toggle.setAttribute('aria-pressed', currentTheme === 'dark' ? 'true' : 'false');
		toggle.innerHTML = currentTheme === 'dark'
			? '<i class="fas fa-moon mr-2" aria-hidden="true"></i>Dark'
			: '<i class="fas fa-sun mr-2" aria-hidden="true"></i>Light';
	}
}

function initTheme() {
	try {
		const saved = localStorage.getItem(THEME_KEY);
		if (saved) applyTheme(saved);
	} catch(e) { /* ignore storage errors */ }
}

function toggleTheme() {
	const next = currentTheme === 'dark' ? 'light' : 'dark';
	applyTheme(next);
	try { localStorage.setItem(THEME_KEY, next); } catch(e) {}
}

// Focus ring: only show outline when navigating by keyboard.
function initFocusRing() {
	function handleMouse() { document.documentElement.classList.add('no-focus-outline'); }
	function handleKey(e) { if (e.key === 'Tab') document.documentElement.classList.remove('no-focus-outline'); }
	document.addEventListener('mousedown', handleMouse);
	document.addEventListener('keydown', handleKey);
	document.documentElement.classList.add('no-focus-outline');
}

// Mobile navigation menu: toggle visibility, manage aria-expanded.
function initMobileNav() {
	const btn = document.getElementById('mobileNavButton');
	const menu = document.getElementById('mobileNavMenu');
	if (!btn || !menu) return;
	// Hidden by default on desktop via CSS; ensure ARIA state sync
	function setState(open) {
		if (open) {
			menu.classList.remove('hidden');
			btn.setAttribute('aria-expanded', 'true');
			announce('Navigation menu opened');
		} else {
			menu.classList.add('hidden');
			btn.setAttribute('aria-expanded', 'false');
			announce('Navigation menu closed');
		}
	}
	btn.addEventListener('click', () => {
		const isHidden = menu.classList.contains('hidden');
		setState(isHidden);
	});
	// Close on Escape
	document.addEventListener('keydown', e => {
		if (e.key === 'Escape' && !menu.classList.contains('hidden')) {
			setState(false);
			btn.focus();
		}
	});
	// Outside click close
	document.addEventListener('click', e => {
		if (!menu.classList.contains('hidden')) {
			if (!menu.contains(e.target) && e.target !== btn) {
				setState(false);
			}
		}
	});
}

// Persist last active demo tab (beta). Stores data-tab value.
const LAST_TAB_KEY = 'gauth-last-tab';
function initTabPersistence() {
	const nav = document.querySelector('nav[aria-label="Interactive demo tabs"]');
	if (!nav) return;
	const buttons = Array.from(nav.querySelectorAll('button[data-tab]'));
	if (!buttons.length) return;
	// Restore
	try {
		const last = localStorage.getItem(LAST_TAB_KEY);
		if (last) {
			const targetBtn = buttons.find(b => b.getAttribute('data-tab') === last);
			const targetPanel = document.getElementById(last);
			if (targetBtn && targetPanel) {
				buttons.forEach(b => {
					b.classList.toggle('active', b === targetBtn);
					b.setAttribute('aria-selected', b === targetBtn ? 'true' : 'false');
					b.setAttribute('tabindex', b === targetBtn ? '0' : '-1');
				});
				document.querySelectorAll('.tab-content').forEach(p => {
					const show = p.id === last;
					p.classList.toggle('active', show);
					p.classList.toggle('hidden', !show);
					if (show) p.removeAttribute('hidden'); else p.setAttribute('hidden','hidden');
				});
			}
		}
	} catch(e) { /* ignore */ }
	// Save on clicks
	buttons.forEach(b => b.addEventListener('click', () => {
		try { localStorage.setItem(LAST_TAB_KEY, b.getAttribute('data-tab')); } catch(e) {}
	}));
}

export function uiInit() {
	initTheme();
	initFocusRing();
	initMobileNav();
	initTabPersistence();
	const themeBtn = document.getElementById('themeToggle');
	if (themeBtn) themeBtn.addEventListener('click', toggleTheme);
	console.info('[ui] initialized (theme=' + currentTheme + ')');
}

// Auto-init when DOM ready.
if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', () => uiInit());
} else {
	uiInit();
}

// Export for other modules (main.js may call uiInit if needed).
export { applyTheme, toggleTheme };

// Simple aria-live announcer (polite) for dynamic state changes.
function announce(msg){
	let region = document.getElementById('ui-live-region');
	if(!region){
		region = document.createElement('div');
		region.id = 'ui-live-region';
		region.setAttribute('aria-live','polite');
		region.className = 'sr-only';
		document.body.appendChild(region);
	}
	region.textContent = msg;
}
