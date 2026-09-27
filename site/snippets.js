// Tabs and copy buttons for the quick start snippets.
//
// Without JavaScript every panel stays visible, which is still usable: the
// panels are only hidden once this script has run and can reveal them again.
(function () {
  "use strict";

  function panelsOf(snippet) {
    return Array.prototype.slice.call(snippet.querySelectorAll("pre[data-panel]"));
  }

  function tabsOf(snippet) {
    return Array.prototype.slice.call(snippet.querySelectorAll(".snippet-tab"));
  }

  function select(snippet, name) {
    panelsOf(snippet).forEach(function (panel) {
      panel.hidden = panel.dataset.panel !== name;
    });
    tabsOf(snippet).forEach(function (tab) {
      var active = tab.dataset.panel === name;
      tab.setAttribute("aria-selected", active ? "true" : "false");
      tab.tabIndex = active ? 0 : -1;
    });
  }

  function visibleText(snippet) {
    var panel = panelsOf(snippet).find(function (candidate) {
      return !candidate.hidden;
    });
    if (!panel) {
      return "";
    }
    // Comment lines explain the command; they are not part of what someone
    // pasting into a shell wants.
    return Array.prototype.slice
      .call(panel.querySelectorAll(".cmd"))
      .map(function (line) {
        return line.textContent;
      })
      .join("\n");
  }

  function setupCopy(snippet) {
    var button = snippet.querySelector(".snippet-copy");
    if (!button || !navigator.clipboard) {
      if (button) {
        button.hidden = true;
      }
      return;
    }

    var reset;
    button.addEventListener("click", function () {
      navigator.clipboard.writeText(visibleText(snippet)).then(
        function () {
          button.textContent = "Copied";
          button.classList.add("is-copied");
        },
        function () {
          button.textContent = "Press Ctrl+C";
        }
      );
      window.clearTimeout(reset);
      reset = window.setTimeout(function () {
        button.textContent = "Copy";
        button.classList.remove("is-copied");
      }, 1600);
    });
  }

  function setupTabs(snippet) {
    var tabs = tabsOf(snippet);
    if (tabs.length === 0) {
      return;
    }

    tabs.forEach(function (tab, index) {
      tab.addEventListener("click", function () {
        select(snippet, tab.dataset.panel);
      });
      // Left and right move between tabs, which is what a tablist should do.
      tab.addEventListener("keydown", function (event) {
        var step = event.key === "ArrowRight" ? 1 : event.key === "ArrowLeft" ? -1 : 0;
        if (step === 0) {
          return;
        }
        event.preventDefault();
        var next = tabs[(index + step + tabs.length) % tabs.length];
        select(snippet, next.dataset.panel);
        next.focus();
      });
    });

    select(snippet, tabs[0].dataset.panel);
  }

  document.querySelectorAll("[data-snippet]").forEach(function (snippet) {
    setupTabs(snippet);
    setupCopy(snippet);
  });
})();
