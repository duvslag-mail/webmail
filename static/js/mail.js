document.addEventListener("htmx:beforeRequest", (e) => {
  const row = e.detail.elt.closest(".message-row");
  if (!row) return; // not a message select thing

  document
    .querySelector(".message-row[aria-current]")
    ?.removeAttribute("aria-current");
  row.setAttribute("aria-current", "true");
});

// mobile back button
document.addEventListener("click", (e) => {
  if (!e.target.closest(".message-view-mobile-back")) return;

  document.querySelector("#reader").innerHTML =
    '<p class="mail-reader-empty">Select a message to read</p>';
  document
    .querySelector(".message-row[aria-current]")
    ?.removeAttribute("aria-current");
});
