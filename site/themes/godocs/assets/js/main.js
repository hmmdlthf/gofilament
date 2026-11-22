// godocs: theme toggle, copy buttons, mobile nav, docs version switcher.
(function () {
  var root = document.documentElement;

  // Theme: stored choice, else the system setting.
  var toggle = document.querySelector("[data-theme-toggle]");
  if (toggle) {
    toggle.addEventListener("click", function () {
      var dark = root.dataset.theme
        ? root.dataset.theme === "dark"
        : window.matchMedia("(prefers-color-scheme: dark)").matches;
      root.dataset.theme = dark ? "light" : "dark";
      try { localStorage.setItem("theme", root.dataset.theme); } catch (e) {}
    });
  }

  // Copy buttons: install box plus every code block.
  function copy(btn, text) {
    var done = function () {
      var old = btn.textContent;
      btn.textContent = "Copied";
      setTimeout(function () { btn.textContent = old; }, 1400);
    };
    if (navigator.clipboard) navigator.clipboard.writeText(text).then(done, function () {});
  }
  document.querySelectorAll("[data-copy]").forEach(function (b) {
    b.addEventListener("click", function () { copy(b, b.dataset.copy); });
  });
  document.querySelectorAll(".highlight").forEach(function (h) {
    var pre = h.querySelector("pre");
    if (!pre) return;
    var b = document.createElement("button");
    b.type = "button";
    b.className = "copy";
    b.textContent = "Copy";
    b.setAttribute("aria-label", "Copy code");
    b.addEventListener("click", function () { copy(b, pre.innerText.replace(/\n$/, "")); });
    h.appendChild(b);
  });

  // Mobile docs navigation.
  var navBtn = document.querySelector("[data-nav-toggle]");
  var backdrop = document.querySelector("[data-nav-close]");
  function setNav(open) {
    document.body.classList.toggle("nav-open", open);
    if (navBtn) navBtn.setAttribute("aria-expanded", open ? "true" : "false");
    if (backdrop) backdrop.hidden = !open;
  }
  if (navBtn) navBtn.addEventListener("click", function () { setNav(!document.body.classList.contains("nav-open")); });
  if (backdrop) backdrop.addEventListener("click", function () { setNav(false); });
  document.addEventListener("keydown", function (e) { if (e.key === "Escape") setNav(false); });
  var current = document.querySelector(".doc-nav a[aria-current]");
  if (current) current.scrollIntoView({ block: "center" });

  // Same page in another version, or that version's home when the page
  // does not exist there (renamed or added later).
  function samePageIn(version) {
    var m = location.pathname.match(/^(.*?\/docs\/)[^/]+\/(.*)$/);
    if (!m) return Promise.resolve(null);
    var home = m[1] + version + "/";
    var target = home + m[2];
    if (target === home) return Promise.resolve(home);
    return fetch(target, { method: "HEAD" })
      .then(function (r) { return r.ok ? target : home; })
      .catch(function () { return home; })
      .then(function (u) { return u + location.hash; });
  }
  var select = document.querySelector("[data-version-switch]");
  if (select) {
    select.addEventListener("change", function () {
      samePageIn(select.value).then(function (u) { if (u) location.href = u; });
    });
  }
  var latestLink = document.querySelector("[data-latest-link]");
  if (latestLink) {
    samePageIn("latest").then(function (u) { if (u) latestLink.href = u; });
  }

  // Highlight the visible section in "On this page".
  var tocLinks = document.querySelectorAll(".toc a[href^='#']");
  if (tocLinks.length && "IntersectionObserver" in window) {
    var byId = {};
    tocLinks.forEach(function (a) { byId[decodeURIComponent(a.hash.slice(1))] = a; });
    var obs = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) {
        if (!en.isIntersecting) return;
        tocLinks.forEach(function (a) { a.classList.remove("active"); });
        var a = byId[en.target.id];
        if (a) a.classList.add("active");
      });
    }, { rootMargin: "-70px 0px -70% 0px" });
    Object.keys(byId).forEach(function (id) {
      var el = document.getElementById(id);
      if (el) obs.observe(el);
    });
  }
})();
