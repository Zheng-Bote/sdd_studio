// Shows the state the shell passes in the query string; the page has no access to Tauri APIs.
const params = new URLSearchParams(location.search);
const texts = {
  starting: 'sdd-desk wird gestartet …',
  choose: 'Kein Projekt gewählt.',
  error: 'sdd-desk konnte nicht gestartet werden.',
};
const state = params.get('state') ?? 'starting';
const status = document.getElementById('status');
status.textContent = texts[state] ?? texts.starting;
status.classList.toggle('error', state === 'error');

const project = params.get('project');
if (project) {
  document.getElementById('project').textContent = `Projekt: ${project}`;
}
const message = params.get('message');
if (message) {
  const log = document.getElementById('log');
  log.textContent = message;
  log.hidden = false;
}
