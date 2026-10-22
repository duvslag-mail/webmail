document.addEventListener("htmx:beforeRequest", (e) => {
  const row = e.detail.elt.closest(".message-row");
  if (!row) return; // not a message select thing

  document
    .querySelector(".message-row[aria-current]")
    ?.removeAttribute("aria-current");
  row.setAttribute("aria-current", "true");
});
