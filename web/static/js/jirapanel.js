document.addEventListener("DOMContentLoaded", function () {
  const DEBUG_KEY = "jirapanel-debug";

  /**
   * Replaces the content of a card by fetching the latest HTML.
   */
  function reloadCard(id, card) {
    if (!card) return;
    fetch(`/api/v1/cell/${id}`)
      .then((res) => res.text())
      .then((html) => {
        card.innerHTML = html;

        // Re-initialize Tablesort on newly inserted tables
        card.querySelectorAll("table.tablesort").forEach((table) => {
          new Tablesort(table);
        });

        // Re-apply debug label if in debug mode
        if (document.body.classList.contains("debug-mode")) {
          injectDebugLabel(card);
        }
      })
      .catch((err) => {
        card.innerHTML = `<div class="alert alert-danger">Error loading cell: ${err}</div>`;
      });
  }

  /**
   * Adds debug label overlay to a card.
   */
  function injectDebugLabel(card) {
    const row = card.getAttribute("data-row");
    const col = card.getAttribute("data-col");
    const span = card.getAttribute("data-col-span");
    const tmpl = card.getAttribute("data-template");

    const label = document.createElement("div");
    label.className = "debug-label";
    label.innerHTML = `row: ${row}\ncol: ${col}\nspan: ${span}\ntemplate: ${tmpl}`;

    card.appendChild(label);
  }

  /**
   * Toggles debug mode on body and manages overlays.
   */
  function toggleDebug() {
    const enabled = document.body.classList.toggle("debug-mode");

    if (enabled) {
      cards.forEach(injectDebugLabel);
      localStorage.setItem(DEBUG_KEY, "1");
    } else {
      document.querySelectorAll(".debug-label").forEach((el) => el.remove());
      localStorage.removeItem(DEBUG_KEY);
    }
  }

  /**
   * Apply debug mode if saved in localStorage.
   */
  function initializeDebugMode() {
    if (localStorage.getItem(DEBUG_KEY) === "1") {
      document.body.classList.add("debug-mode");
      cards.forEach(injectDebugLabel);
    }
  }

  /**
   * Main refresh loop: checks config and cell hashes, and reloads as needed.
   */
  function refresh() {
    fetch("/api/v1/hash/config")
      .then((res) => res.text())
      .then((newConfigHash) => {
        if (newConfigHash !== configHash) {
          location.reload();
          return;
        }

        cards.forEach((card) => {
          const id = card.getAttribute("data-cell-id");
          fetch(`/api/v1/hash/${id}`)
            .then((res) => res.text())
            .then((newHash) => {
              if (!cellHashes[id] || cellHashes[id] !== newHash) {
                cellHashes[id] = newHash;
                reloadCard(id, card);
              }
            })
            .catch((err) => {
              console.warn("Failed to check hash for cell", id, err);
            });
        });
      })
      .catch((err) => {
        console.warn("Failed to check config hash", err);
      });
  }

  // Handle debug toggle via keypress
  document.addEventListener("keydown", function (e) {
    if (e.key === "d" || e.key === "D") toggleDebug();
  });

  // Initial card load
  const cards = document.querySelectorAll("[data-cell-id]");
  cards.forEach((card) => {
    const id = card.getAttribute("data-cell-id");
    reloadCard(id, card);
  });

  // Refresh loop
  const meta = document.querySelector('meta[name="refresh-interval"]');
  const interval = meta ? parseInt(meta.content) : 60;
  setInterval(refresh, interval * 1000);

  // Apply debug state if persisted
  initializeDebugMode();
});
