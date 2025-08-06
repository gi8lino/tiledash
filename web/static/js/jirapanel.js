document.addEventListener("DOMContentLoaded", function () {
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
      })
      .catch((err) => {
        card.innerHTML = `<div class="alert alert-danger">Error loading cell: ${err}</div>`;
      });
  }

  /**
   * Main refresh loop: checks config and cell hashes, and reloads as needed.
   */
  function refresh() {
    // First: check config hash
    fetch("/api/v1/hash/config")
      .then((res) => res.text())
      .then((newConfigHash) => {
        if (newConfigHash !== configHash) {
          // Reload full page if base config changed
          location.reload();
          return; // prevent card checks
        }

        // Otherwise, check individual cells
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

  // Initial card load
  const cards = document.querySelectorAll("[data-cell-id]");
  cards.forEach((card) => {
    const id = card.getAttribute("data-cell-id");
    reloadCard(id, card);
  });

  // Start refresh timer using value from <meta name="refresh-interval">
  const meta = document.querySelector('meta[name="refresh-interval"]');
  const interval = meta ? parseInt(meta.content) : 60;
  setInterval(refresh, interval * 1000);
});
