document.addEventListener("DOMContentLoaded", function () {
  const DEBUG_KEY = "tiledash-debug";
  const configHash =
    document.querySelector('meta[name="config-hash"]')?.content || "";

  // Track all card elements
  const cards = document.querySelectorAll("[data-tile-id]");
  // Track in-flight reloads to prevent double-fetching
  const inFlight = new Set();

  /**
   * Replaces the content of a card by fetching the latest HTML.
   */
  function reloadCard(id, card) {
    if (!card || inFlight.has(id)) return;
    inFlight.add(id);

    fetch(`/api/v1/tile/${id}`)
      .then((res) => res.text())
      .then((html) => {
        card.innerHTML = html;

        // Hide/show card based on template signal
        const isHidden = !!card.querySelector("[data-td-hidden]");
        card.classList.toggle("td-hidden", isHidden);

        // Re-init Tablesort on newly inserted tables
        card.querySelectorAll("table.tablesort").forEach((table) => {
          // eslint-disable-next-line no-undef
          new Tablesort(table);
        });

        // Re-apply debug label if in debug mode
        if (document.body.classList.contains("debug-mode")) {
          injectDebugLabel(card);
        }
      })
      .catch((err) => {
        // Only for true network failures; server normally returns HTML (incl. errors)
        card.innerHTML = `
          <div class="alert alert-danger" role="alert">
            <strong>Network error:</strong> Could not load tile.
            <div><small>${String(err)}</small></div>
          </div>
        `;
        // Keep hidden state off on network error so you can see the problem
        card.classList.remove("td-hidden");
      })
      .finally(() => {
        inFlight.delete(id);
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
    const hidden = card.classList.contains("td-hidden") ? " (hidden)" : "";

    // Avoid stacking duplicates
    const existing = card.querySelector(".debug-label");
    if (existing) existing.remove();

    const label = document.createElement("div");
    label.className = "debug-label";
    label.innerHTML = `row: ${row} | col: ${col} | span: ${span}${hidden}\ntemplate: ${tmpl}`;
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
   * Main refresh loop: checks config and tile hashes, and reloads as needed.
   */
  function refresh() {
    fetch("/api/v1/hash/config")
      .then((res) => res.text())
      .then((newConfigHash) => {
        if (newConfigHash !== configHash) {
          if (document.body.classList.contains("debug-mode")) {
            console.log("Config hash changed. Reloading page...");
          }
          location.reload();
          return;
        }

        cards.forEach((card) => {
          const id = card.getAttribute("data-tile-id");
          fetch(`/api/v1/hash/${id}`)
            .then((res) => res.text())
            .then((newHash) => {
              if (!tileHashes[id] || tileHashes[id] !== newHash) {
                if (document.body.classList.contains("debug-mode")) {
                  console.log(
                    `Reloading tile ${id} (old=${tileHashes[id]}, new=${newHash})`,
                  );
                }
                tileHashes[id] = newHash;
                reloadCard(id, card);
              } else if (document.body.classList.contains("debug-mode")) {
                console.log(`Cell ${id} unchanged (hash=${newHash})`);
              }
            })
            .catch((err) => {
              console.warn("Failed to check hash for tile", id, err);
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
  cards.forEach((card) => {
    const id = card.getAttribute("data-tile-id");
    reloadCard(id, card);
  });

  // Refresh loop
  const meta = document.querySelector('meta[name="refresh-interval"]');
  const interval = meta ? parseInt(meta.content, 10) : 60;
  setInterval(refresh, interval * 1000);

  // Restore debug state
  initializeDebugMode();
});
