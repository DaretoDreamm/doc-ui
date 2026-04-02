package doctemplates

// ProseCSS returns shared CSS using palette + typography CSS variables.
func ProseCSS() string {
	return `
				body { font-family: var(--font-body); background: var(--c-bg); color: var(--c-text); }
				h1, h2, h3, h4, h5, h6 { font-family: var(--font-display); }
				code, pre { font-family: var(--font-code); }

				/* Themed scrollbar */
				* { scrollbar-width: thin; scrollbar-color: var(--c-border) transparent; }
				::-webkit-scrollbar { width: 5px; height: 5px; }
				::-webkit-scrollbar-track { background: transparent; }
				::-webkit-scrollbar-thumb { background: var(--c-border); border-radius: 99px; }
				::-webkit-scrollbar-thumb:hover { background: var(--c-primary); }

				::selection { background: var(--c-active-bg); color: var(--c-active-text); }
				:focus-visible { outline: 2px solid var(--c-primary); outline-offset: 2px; }
				input:focus { outline: none; }

				.prose h1 { font-size: 2em; font-weight: 800; margin-bottom: 0.5em; letter-spacing: -0.03em; color: var(--c-text); }
				.prose h2 { font-size: 1.4em; font-weight: 700; margin-top: 2em; margin-bottom: 0.6em; padding-bottom: 0.4em; border-bottom: 1px solid var(--c-border); color: var(--c-text); }
				.prose h3 { font-size: 1.15em; font-weight: 700; margin-top: 1.5em; margin-bottom: 0.5em; color: var(--c-text); }
				.prose p { margin-bottom: 1.15em; line-height: 1.75; color: var(--c-text-muted); }
				.prose pre { background: var(--c-code-bg); color: var(--c-code-text); padding: 1em 1.2em; border-radius: 8px; overflow-x: auto; margin-bottom: 1.25em; font-size: 0.8125em; line-height: 1.7; border: 1px solid var(--c-border); }
				.prose code { font-size: 0.8125em; background: var(--c-active-bg); padding: 0.15em 0.35em; border-radius: 4px; color: var(--c-primary); }
				.prose pre code { background: none; padding: 0; color: inherit; border: none; border-radius: 0; }
				.prose ul, .prose ol { margin-bottom: 1.15em; padding-left: 1.5em; }
				.prose li { margin-bottom: 0.35em; line-height: 1.7; color: var(--c-text-muted); }
				.prose a { color: var(--c-link); text-decoration: underline; text-underline-offset: 2px; text-decoration-color: var(--c-border); transition: text-decoration-color 0.15s; }
				.prose a:hover { text-decoration-color: var(--c-link); }
				.prose table { width: 100%; border-collapse: collapse; margin-bottom: 1.25em; font-size: 0.9em; }
				.prose th, .prose td { border: 1px solid var(--c-border); padding: 0.5em 0.75em; text-align: left; }
				.prose th { background: var(--c-sidebar); font-weight: 600; color: var(--c-text); }
				.prose td { color: var(--c-text-muted); }
				.prose blockquote { border-left: 3px solid var(--c-blockquote); padding: 0.75em 1em; color: var(--c-text-muted); margin-bottom: 1.15em; background: var(--c-sidebar); border-radius: 0 6px 6px 0; }
				.prose img { max-width: 100%; height: auto; border-radius: 8px; }
				.prose hr { border: none; border-top: 1px solid var(--c-border); margin: 2em 0; }

				/* Spotlight search */
				.spotlight-overlay { position: fixed; inset: 0; z-index: 999; display: flex; align-items: flex-start; justify-content: center; padding-top: 15vh; background: rgba(0,0,0,0.5); backdrop-filter: blur(4px); }
				.spotlight-modal { width: 100%; max-width: 560px; border-radius: 12px; overflow: hidden; border: 1px solid var(--c-border); background: var(--c-surface); box-shadow: 0 25px 50px rgba(0,0,0,0.3); }
				.spotlight-header { display: flex; align-items: center; gap: 10px; padding: 12px 16px; border-bottom: 1px solid var(--c-border); }
				.spotlight-icon { width: 18px; height: 18px; color: var(--c-text-muted); flex-shrink: 0; }
				.spotlight-input { flex: 1; background: none; border: none; color: var(--c-text); font-size: 15px; font-family: inherit; }
				.spotlight-input::placeholder { color: var(--c-text-muted); opacity: 0.6; }
				.spotlight-input:focus { outline: none; }
				.spotlight-kbd { font-family: 'Fira Code', monospace; font-size: 10px; padding: 2px 6px; border-radius: 4px; border: 1px solid var(--c-border); color: var(--c-text-muted); background: var(--c-bg); }
				.spotlight-results { max-height: 360px; overflow-y: auto; }
				.spotlight-result { border-bottom: 1px solid var(--c-border); }
				.spotlight-result:last-child { border-bottom: none; }
				.spotlight-result:hover, .spotlight-result.active { background: var(--c-active-bg); }

				/* Spotlight trigger button */
				.spotlight-trigger { display: flex; align-items: center; gap: 8px; padding: 6px 12px; border-radius: 8px; border: 1px solid var(--c-border); color: var(--c-text-muted); background: var(--c-bg); cursor: pointer; transition: border-color 0.15s, color 0.15s; font-family: inherit; font-size: 13px; }
				.spotlight-trigger:hover { border-color: var(--c-primary); color: var(--c-text); }
				.spotlight-trigger-label { }
				.spotlight-trigger-kbd { display: flex; align-items: center; gap: 1px; font-family: 'Fira Code', monospace; font-size: 10px; padding: 1px 5px; border-radius: 4px; border: 1px solid var(--c-border); color: var(--c-text-muted); background: var(--c-surface); margin-left: 4px; }
	`
}

// SpotlightJS returns the JavaScript for spotlight functionality.
func SpotlightJS() string {
	return `
		function openSpotlight() {
			const el = document.getElementById('spotlight');
			el.style.display = 'flex';
			const input = document.getElementById('spotlight-input');
			input.value = '';
			input.focus();
			document.getElementById('spotlight-results').innerHTML = '';
		}
		function closeSpotlight() {
			document.getElementById('spotlight').style.display = 'none';
		}
		document.addEventListener('keydown', function(e) {
			if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
				e.preventDefault();
				const el = document.getElementById('spotlight');
				if (el.style.display === 'none') openSpotlight(); else closeSpotlight();
			}
			if (e.key === 'Escape') closeSpotlight();

			// Keyboard navigation in results
			if (document.getElementById('spotlight').style.display !== 'none') {
				const results = document.querySelectorAll('.spotlight-result');
				if (results.length === 0) return;
				const active = document.querySelector('.spotlight-result.active');
				let idx = active ? Array.from(results).indexOf(active) : -1;

				if (e.key === 'ArrowDown') {
					e.preventDefault();
					if (active) active.classList.remove('active');
					idx = (idx + 1) % results.length;
					results[idx].classList.add('active');
					results[idx].scrollIntoView({block: 'nearest'});
				} else if (e.key === 'ArrowUp') {
					e.preventDefault();
					if (active) active.classList.remove('active');
					idx = idx <= 0 ? results.length - 1 : idx - 1;
					results[idx].classList.add('active');
					results[idx].scrollIntoView({block: 'nearest'});
				} else if (e.key === 'Enter' && active) {
					e.preventDefault();
					window.location.href = active.getAttribute('href');
				}
			}
		});
	`
}
