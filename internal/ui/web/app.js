"use strict";
// SDD Studio UI. Every text is set via textContent; nothing from the server is ever inserted as HTML.

const TOKEN_KEY = "sdd-token";
(function takeToken() {
  const url = new URL(location.href);
  const t = url.searchParams.get("token");
  if (t) {
    try { sessionStorage.setItem(TOKEN_KEY, t); } catch (e) { window.__sddToken = t; }
    url.searchParams.delete("token");
    history.replaceState(null, "", url.pathname + url.hash);
  }
})();
function token() {
  try { return sessionStorage.getItem(TOKEN_KEY) || window.__sddToken || ""; } catch (e) { return window.__sddToken || ""; }
}

function h(tag, attrs, ...children) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v === undefined || v === null || v === false) continue;
    if (k === "class") el.className = v;
    else if (k.startsWith("on")) el.addEventListener(k.slice(2), v);
    else el.setAttribute(k, v === true ? "" : v);
  }
  for (const c of children.flat()) {
    if (c === null || c === undefined || c === false) continue;
    el.append(c instanceof Node ? c : document.createTextNode(String(c)));
  }
  return el;
}

async function getJSON(url) {
  const res = await fetch(url, { cache: "no-store" });
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body.error || res.statusText);
  return body;
}
async function postJSON(url, data) {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-SDD-Token": token() },
    body: JSON.stringify(data),
  });
  const body = res.status === 204 ? {} : await res.json().catch(() => ({}));
  if (!res.ok) { const err = new Error(body.error || res.statusText); err.body = body; throw err; }
  return body;
}

const badge = (kind, label) => h("span", { class: "badge " + kind }, label ?? kind);
const notice = (kind, ...text) => h("div", { class: "notice " + kind }, ...text);
const handbookLink = (url, label) => {
  const anchor = (url || "").split("#")[1];
  return anchor ? h("a", { href: "/handbook#" + anchor, target: "_blank", rel: "noopener" }, label) : label;
};

let meta = { project: "", root: "", editor: false, gates: [] };
let poll = null;
const view = document.getElementById("view");

// ---------- Views ----------

const views = {
  overview: { title: "Übersicht", key: "1", render: renderOverview },
  artifacts: { title: "Artefakte", key: "2", render: renderArtifacts },
  findings: { title: "Befunde", key: "3", render: renderFindings },
  new: { title: "Neues Artefakt", key: "4", render: renderNew },
  adopt: { title: "Übernahme", key: "5", render: renderAdopt },
};

async function renderOverview() {
  const o = await getJSON("/api/overview");
  const m = o.maturity || {};
  const gates = [{ gate: "lint", counts: o.lint }].concat(o.gates || []);
  return [
    h("h1", {}, "Übersicht"),
    h("p", { class: "lead" }, "Profil ", h("code", {}, o.profile), " · entspricht ", h("code", {}, "sdd report")),
    h("div", { class: "cards" },
      h("div", { class: "card big" }, h("div", { class: "label" }, "Reifegrad"), h("div", { class: "value" }, "L" + m.level)),
      h("div", { class: "card" }, h("div", { class: "label" }, "Offene Befunde (lint)"), h("div", { class: "value" }, String((o.findings || []).length))),
      h("div", { class: "card" }, h("div", { class: "label" }, "Baseline-Einträge"), h("div", { class: "value" }, String(o.debt_now)),
        h("div", { class: "muted" }, o.since_error ? "Trend nicht ermittelbar" : "vorher " + o.debt_before + " (" + o.since + ")")),
    ),
    m.next ? notice("warn", h("strong", {}, "Nächster Schritt (L" + m.next.level + "): "), m.next.advice) : notice("ok", "Alle Kriterien bis L4 sind erfüllt."),
    (o.added || []).length ? notice("error", h("strong", {}, "Verstoß gegen die Ratchet-Regel: "), o.added.length + " neue Baseline-Einträge gegenüber " + o.since + ".") : null,
    h("h2", {}, "Gates"),
    h("div", { class: "cards" }, gates.map((g) => h("div", { class: "card" },
      h("div", { class: "label" }, g.gate),
      g.error ? h("div", { class: "muted" }, "nicht ausführbar: " + g.error)
        : h("div", { class: "counts" }, badge(g.counts.error ? "error" : "ok", g.counts.error ? "rot" : "grün"), " ",
          h("span", {}, g.counts.error + " error · " + g.counts.warn + " warn · " + g.counts.baselined + " baselined"))))),
    h("h2", {}, "Reifegrad-Kriterien"),
    h("table", {}, h("thead", {}, h("tr", {}, h("th", {}, "Stufe"), h("th", {}, "Kriterium"), h("th", {}, "Erfüllt"), h("th", {}, "Befund"))),
      h("tbody", {}, (m.criteria || []).map((c) => h("tr", {}, h("td", {}, "L" + c.level), h("td", {}, c.name),
        h("td", {}, badge(c.met ? "ok" : "warn", c.met ? "ja" : "nein")), h("td", {}, c.detail))))),
    (o.debt || []).length ? [h("h2", {}, "Schuldenstand je Regel"),
      h("table", {}, h("thead", {}, h("tr", {}, h("th", {}, "Regel"), h("th", {}, "Gate"), h("th", {}, "jetzt"), h("th", {}, "vorher"))),
        h("tbody", {}, o.debt.map((d) => h("tr", {}, h("td", {}, d.rule_id), h("td", {}, d.gate), h("td", { class: "num" }, String(d.now)), h("td", { class: "num" }, String(d.before))))))] : null,
  ];
}

async function renderArtifacts() {
  const rows = await getJSON("/api/artifacts");
  const today = new Date().toISOString().slice(0, 10);
  const filter = h("input", { type: "text", placeholder: "Filtern nach Pfad, Typ, Feature …", "aria-label": "Artefakte filtern" });
  const tbody = h("tbody");
  const draw = () => {
    const q = filter.value.toLowerCase();
    tbody.replaceChildren(...rows
      .filter((r) => !q || [r.path, r.kind, r.feature, r.status].join(" ").toLowerCase().includes(q))
      .map((r) => {
        const overdue = r.archive_after && !r.archived && r.archive_after < today;
        return h("tr", {},
          h("td", {}, r.kind + (r.archived ? " · Archiv" : "")),
          h("td", { class: "path" }, r.path),
          h("td", {}, r.feature || "–"),
          h("td", {}, r.status ? badge(["archived", "superseded"].includes(r.status) ? "neutral" : "ok", r.status) : "–"),
          h("td", { class: "num" }, r.age_days + " d"),
          h("td", {}, r.archive_after ? (overdue ? badge("warn", r.archive_after + " überschritten") : r.archive_after) : "–"));
      }));
  };
  filter.addEventListener("input", draw);
  draw();
  return [
    h("h1", {}, "Artefakte"),
    h("p", { class: "lead" }, rows.length + " Markdown-Dateien, zugeordnet zu den sechs Artefakttypen (wie ", h("code", {}, "sdd adopt"), ")."),
    h("div", { class: "toolbar" }, filter),
    h("table", {}, h("thead", {}, h("tr", {}, ["Typ", "Pfad", "Feature", "Status", "Alter", "Archivfrist"].map((t) => h("th", {}, t)))), tbody),
  ];
}

const findingsState = { gate: "", status: "all" };
async function renderFindings() {
  const gateSel = h("select", { "aria-label": "Gate" }, h("option", { value: "" }, "lint (ohne Gate)"),
    meta.gates.map((g) => h("option", { value: g }, g)));
  gateSel.value = findingsState.gate;
  const chips = h("div", { class: "chips", role: "group", "aria-label": "Status" });
  const live = h("span", { class: "live" }, "live");
  const summary = h("span", { class: "muted" });
  const tbody = h("tbody");
  let report = null;

  const visible = () => (report ? report.findings : []).filter((f) =>
    findingsState.status === "all" || (findingsState.status === "new" ? f.status !== "baselined" : f.status === "baselined"));
  const draw = () => {
    for (const b of chips.children) b.classList.toggle("on", b.dataset.status === findingsState.status);
    const list = visible();
    const s = report.summary;
    summary.textContent = s.error + " error · " + s.warn + " warn · " + s.info + " info · " + s.suppressed + " baselined";
    tbody.replaceChildren(...(list.length ? list.map((f) => h("tr", {},
      h("td", {}, badge(f.severity), f.status === "baselined" ? h("div", { class: "muted" }, "Altbestand") : null),
      h("td", { class: "nowrap" }, handbookLink(f.handbook_url, f.rule_id)),
      h("td", { class: "path" }, h("a", { href: "vscode://file/" + meta.root + "/" + f.path + ":" + (f.line || 1), title: "In VS Code öffnen" }, f.path + ":" + f.line),
        meta.editor ? [" ", h("button", { class: "link", onclick: () => openInEditor(f) }, "$EDITOR")] : null),
      h("td", {}, f.message))) : [h("tr", {}, h("td", { colspan: "4", class: "muted" }, "Keine Befunde."))]));
  };
  const load = async () => {
    try {
      report = await getJSON("/api/findings" + (findingsState.gate ? "?gate=" + encodeURIComponent(findingsState.gate) : ""));
      live.classList.remove("stale");
      draw();
    } catch (e) {
      live.classList.add("stale");
      summary.textContent = "Fehler: " + e.message;
    }
  };
  for (const [key, label] of [["all", "Alle"], ["new", "Neu"], ["baselined", "Altbestand"]]) {
    const b = h("button", { type: "button", onclick: () => { findingsState.status = key; draw(); } }, label);
    b.dataset.status = key;
    chips.append(b);
  }
  gateSel.addEventListener("change", () => { findingsState.gate = gateSel.value; load(); });
  await load();
  poll = setInterval(() => { if (document.visibilityState === "visible") load(); }, 3000);

  const exportAs = (kind) => {
    const list = visible();
    let blob;
    if (kind === "json") blob = new Blob([JSON.stringify(Object.assign({}, report, { findings: list }), null, 2)], { type: "application/json" });
    else {
      const esc = (v) => '"' + String(v ?? "").replaceAll('"', '""') + '"';
      const lines = [["rule_id", "severity", "status", "path", "line", "message"].join(",")]
        .concat(list.map((f) => [f.rule_id, f.severity, f.status, f.path, f.line, f.message].map(esc).join(",")));
      blob = new Blob([lines.join("\n") + "\n"], { type: "text/csv" });
    }
    const a = h("a", { href: URL.createObjectURL(blob), download: "sdd-befunde." + kind });
    document.body.append(a); a.click(); a.remove();
  };
  return [
    h("h1", {}, "Befunde"),
    h("p", { class: "lead" }, "Entspricht ", h("code", {}, "sdd lint --json"), " bzw. ", h("code", {}, "sdd gate <gate> --json"), "; aktualisiert sich alle drei Sekunden."),
    h("div", { class: "toolbar" }, gateSel, chips, live, summary, h("span", { class: "spacer" }),
      h("button", { class: "secondary", onclick: () => exportAs("json") }, "JSON"),
      h("button", { class: "secondary", onclick: () => exportAs("csv") }, "CSV")),
    h("table", {}, h("thead", {}, h("tr", {}, ["Schwere", "Regel", "Fundstelle", "Meldung"].map((t) => h("th", {}, t)))), tbody),
  ];
}

async function openInEditor(f) {
  try { await postJSON("/api/open", { path: f.path, line: f.line || 0 }); }
  catch (e) { alert("Öffnen fehlgeschlagen: " + e.message); }
}

function opDetails(op) {
  const label = { create: "anlegen", prepend: "Frontmatter voranstellen", skip: "übersprungen" }[op.action] || op.action;
  return h("details", { class: "op", open: op.action !== "skip" },
    h("summary", {}, badge(op.action === "skip" ? "neutral" : "ok", label), " ", h("code", {}, op.path), op.reason ? h("span", { class: "muted" }, " — " + op.reason) : null),
    op.content ? h("pre", {}, h("code", {}, op.content)) : null);
}

async function renderNew() {
  const type = h("select", { name: "type" },
    h("option", { value: "feature" }, "Feature: PRD + Tech Spec"),
    h("option", { value: "prd" }, "PRD"), h("option", { value: "design" }, "Tech Spec"), h("option", { value: "plan" }, "Plan"),
    h("option", { value: "adr" }, "ADR"), h("option", { value: "doc" }, "Doku/Exploration"));
  const feature = h("input", { type: "text", placeholder: "billing-export", pattern: "[a-z0-9-]+" });
  const title = h("input", { type: "text", placeholder: "Titel" });
  const slug = h("input", { type: "text", placeholder: "aus dem Titel abgeleitet" });
  const touches = h("input", { type: "text", placeholder: "cmd/**, openapi.yaml" });
  const archiveAfter = h("input", { type: "date" });
  const out = h("div");
  const request = (apply) => {
    const t = type.value;
    return {
      types: t === "feature" ? [] : [t],
      feature: ["adr", "doc"].includes(t) ? "" : feature.value.trim(),
      title: title.value, slug: slug.value,
      touches: touches.value.split(",").map((s) => s.trim()).filter(Boolean),
      archive_after: archiveAfter.value, apply,
    };
  };
  const run = async (apply) => {
    out.replaceChildren();
    try {
      const res = await postJSON("/api/new", request(apply));
      if (!apply) {
        const writes = res.ops.filter((o) => o.action !== "skip").length;
        out.append(h("h2", {}, "Vorschau — es wurde nichts geschrieben"), res.ops.map(opDetails),
          writes ? h("button", { onclick: () => { if (confirm(writes + " Datei(en) anlegen?")) run(true); } }, writes + " Datei(en) anlegen")
            : notice("warn", "Nichts anzulegen."));
      } else {
        out.append(notice("ok", "Angelegt: ", res.results.filter((r) => r.Status !== "skipped").map((r) => r.Path).join(", ") || "nichts"));
      }
    } catch (e) { out.append(notice("error", e.message)); }
  };
  const syncFields = () => {
    const isOther = ["adr", "doc"].includes(type.value);
    feature.disabled = isOther; touches.disabled = isOther; archiveAfter.disabled = isOther; slug.disabled = !isOther;
  };
  type.addEventListener("change", syncFields);
  syncFields();
  return [
    h("h1", {}, "Neues Artefakt"),
    h("p", { class: "lead" }, "Entspricht ", h("code", {}, "sdd new"), ". Zuerst die Vorschau mit Pfad und Inhalt, geschrieben wird erst nach Bestätigung — nur neue Dateien, nie überschreibend."),
    h("form", { class: "grid", onsubmit: (e) => { e.preventDefault(); run(false); } },
      h("label", {}, "Typ"), type,
      h("label", {}, "Feature-Slug"), feature,
      h("label", {}, "Titel"), title,
      h("label", {}, "Slug (ADR/Doku)"), slug,
      h("label", {}, "touches"), touches, h("div", { class: "hint" }, "Erklärte Änderungsabsicht für pre-implementation, kommagetrennte Globs."),
      h("label", {}, "archive-after"), archiveAfter,
      h("div", { class: "actions" }, h("button", { type: "submit" }, "Vorschau"))),
    out,
  ];
}

async function renderAdopt() {
  const a = await getJSON("/api/adopt");
  const branch = h("input", { type: "text", value: a.branch, "aria-label": "Branch" });
  const out = h("div");
  const apply = async () => {
    if (!confirm("Neuen Branch „" + branch.value + "“ anlegen und je Schritt einen Commit erstellen? Der aktuelle Branch bleibt unverändert.")) return;
    out.replaceChildren();
    try {
      const res = await postJSON("/api/adopt/apply", { branch: branch.value });
      out.append(notice("ok", "Angewendet auf Branch ", h("code", {}, res.Branch), " (ausgehend von ", h("code", {}, res.FromBranch), ")."),
        h("ul", {}, (res.Commits || []).map((c) => h("li", {}, h("code", {}, c.Hash), " ", c.Step, " — ", c.Paths.join(", ")))));
    } catch (e) { out.append(notice("error", e.message)); }
  };
  const steps = a.steps.map((s) => [h("h2", {}, s.title), s.ops.map(opDetails)]);
  return [
    h("h1", {}, "Übernahme"),
    h("p", { class: "lead" }, "Probelauf von ", h("code", {}, "sdd adopt"), " — es wurde nichts geschrieben. Reifegrad vor der Übernahme: L" + a.maturity.level + "."),
    (a.conflicts || []).length ? notice("warn", h("strong", {}, "Konflikte: "), a.conflicts.join(" · ")) : null,
    steps.length ? steps : notice("ok", "Keine Datei-Schritte nötig."),
    a.baseline_planned ? notice("warn", "Danach: Baseline und ADR-Lockfile für alle Gates, vor dem ersten Gate-Lauf.") : null,
    h("div", { class: "toolbar" }, h("label", {}, "Branch "), branch, h("button", { onclick: apply }, "Auf neuem Branch anwenden")),
    out,
    h("h2", {}, "Bericht"),
    h("pre", {}, h("code", {}, a.report)),
  ];
}

// ---------- Routing ----------

async function show(name) {
  if (!views[name]) name = "overview";
  if (poll) { clearInterval(poll); poll = null; }
  for (const a of document.querySelectorAll(".sidebar a[data-view]")) a.classList.toggle("active", a.dataset.view === name);
  document.title = views[name].title + " — SDD Studio";
  view.replaceChildren(h("p", { class: "muted" }, "Lade …"));
  try { view.replaceChildren(...[await views[name].render()].flat(3).filter(Boolean)); }
  catch (e) { view.replaceChildren(h("h1", {}, views[name].title), notice("error", e.message)); }
  view.focus({ preventScroll: true });
}
window.addEventListener("hashchange", () => show(location.hash.slice(1)));

// ---------- Command palette ----------

const palette = document.getElementById("palette");
const pInput = document.getElementById("palette-input");
const pList = document.getElementById("palette-list");
const commands = Object.entries(views).map(([name, v]) => ({ label: v.title, hint: v.key, run: () => { location.hash = name; } }))
  .concat([{ label: "Handbuch", hint: "H", run: () => window.open("/handbook", "_blank", "noopener") }]);
let sel = 0;
function drawPalette() {
  const q = pInput.value.toLowerCase();
  const items = commands.filter((c) => c.label.toLowerCase().includes(q));
  sel = Math.min(sel, Math.max(items.length - 1, 0));
  pList.replaceChildren(...items.map((c, i) => h("li", { class: i === sel ? "sel" : "", role: "option", onclick: () => { closePalette(); c.run(); } },
    h("span", {}, c.label), h("kbd", {}, c.hint))));
  return items;
}
function openPalette() { palette.hidden = false; pInput.value = ""; sel = 0; drawPalette(); pInput.focus(); }
function closePalette() { palette.hidden = true; }
pInput.addEventListener("input", () => { sel = 0; drawPalette(); });
pInput.addEventListener("keydown", (e) => {
  const items = drawPalette();
  if (e.key === "ArrowDown") { sel = (sel + 1) % Math.max(items.length, 1); drawPalette(); e.preventDefault(); }
  else if (e.key === "ArrowUp") { sel = (sel - 1 + items.length) % Math.max(items.length, 1); drawPalette(); e.preventDefault(); }
  else if (e.key === "Enter" && items[sel]) { closePalette(); items[sel].run(); }
  else if (e.key === "Escape") closePalette();
});
palette.addEventListener("click", (e) => { if (e.target === palette) closePalette(); });
document.getElementById("palette-open").addEventListener("click", openPalette);
document.addEventListener("keydown", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") { e.preventDefault(); palette.hidden ? openPalette() : closePalette(); return; }
  const typing = ["INPUT", "SELECT", "TEXTAREA"].includes(document.activeElement.tagName);
  if (typing || e.ctrlKey || e.metaKey || e.altKey || !palette.hidden) return;
  const cmd = commands.find((c) => c.hint.toLowerCase() === e.key.toLowerCase());
  if (cmd) { e.preventDefault(); cmd.run(); }
});

(async function start() {
  try {
    meta = await getJSON("/api/meta");
    document.getElementById("project").textContent = meta.project;
  } catch (e) { /* the view shows the error */ }
  show(location.hash.slice(1) || "overview");
})();
