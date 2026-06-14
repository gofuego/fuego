// Client-side filter for search inputs
document.querySelectorAll("[data-search-input]").forEach(input => {
  const list = input.closest("[data-search-list]") || document.querySelector("[data-search-list]");
  if (!list) return;

  const items = list.querySelectorAll("[data-search-item]");
  input.addEventListener("input", () => {
    const q = input.value.toLowerCase();
    items.forEach(item => {
      item.style.display = item.textContent.toLowerCase().includes(q) ? "" : "none";
    });
  });
});

// Collapsible affects patterns
document.querySelectorAll("[data-affects-pattern]").forEach(el => {
  const expanded = el.querySelector("[data-affects-expanded]");
  if (!expanded) return;

  const toggle = el.querySelector("[data-affects-toggle]");
  if (!toggle) return;

  expanded.style.display = "none";
  toggle.addEventListener("click", () => {
    const open = expanded.style.display !== "none";
    expanded.style.display = open ? "none" : "block";
    toggle.textContent = open ? "expand" : "collapse";
  });
});
