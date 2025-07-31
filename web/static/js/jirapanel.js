document.addEventListener("DOMContentLoaded", function () {
  document.querySelectorAll("table.tablesort").forEach(function (table) {
    new Tablesort(table);
  });

  const meta = document.querySelector('meta[name="refresh-interval"]');
  let interval = meta ? parseInt(meta.content) : 60;

  let timer = setInterval(reload, interval * 1000);

  function reload() {
    location.reload(true);
  }

  function updateIntervalIfChanged() {
    const newMeta = document.querySelector('meta[name="refresh-interval"]');
    if (!newMeta) {
      return;
    }
    const newVal = parseInt(newMeta.content);
    if (!isNaN(newVal) && newVal !== interval) {
      clearInterval(timer);
      interval = newVal;
      timer = setInterval(reload, interval * 1000);
    }
  }

  setInterval(updateIntervalIfChanged, 30000); // check every 30s if meta has changed
});
