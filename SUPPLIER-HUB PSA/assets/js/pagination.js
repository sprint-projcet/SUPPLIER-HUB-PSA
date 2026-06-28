(function () {
  const DEFAULT_PAGE_SIZE = 10;
  const DEFAULT_ITEM_LABEL = "data";

  function asArray(items) {
    return Array.isArray(items) ? items : [];
  }

  function clamp(value, min, max) {
    return Math.min(Math.max(value, min), max);
  }

  function getPageSize(state) {
    const pageSize = Number(state && state.pageSize);
    return pageSize > 0 ? pageSize : DEFAULT_PAGE_SIZE;
  }

  function totalPages(totalItems, pageSize) {
    return Math.max(1, Math.ceil(totalItems / pageSize));
  }

  function create(options = {}) {
    return {
      page: 1,
      pageSize: Number(options.pageSize) || DEFAULT_PAGE_SIZE,
      itemLabel: options.itemLabel || DEFAULT_ITEM_LABEL,
    };
  }

  function reset(state) {
    if (state) state.page = 1;
    return state;
  }

  function getPage(items, state) {
    const source = asArray(items);
    const pageSize = getPageSize(state);
    const pages = totalPages(source.length, pageSize);
    const page = clamp(Number(state && state.page) || 1, 1, pages);
    if (state) state.page = page;

    const start = (page - 1) * pageSize;
    const end = Math.min(start + pageSize, source.length);

    return {
      items: source.slice(start, end),
      page,
      pageSize,
      totalItems: source.length,
      totalPages: pages,
      startIndex: source.length === 0 ? 0 : start + 1,
      endIndex: end,
    };
  }

  function resolveContainer(target) {
    if (!target) return null;
    if (typeof target === "string") return document.getElementById(target);
    return target;
  }

  function pageList(currentPage, pages) {
    if (pages <= 5) {
      return Array.from({ length: pages }, (_, index) => index + 1);
    }

    const result = [1];
    const start = Math.max(2, currentPage - 1);
    const end = Math.min(pages - 1, currentPage + 1);

    if (start > 2) result.push("...");
    for (let page = start; page <= end; page += 1) result.push(page);
    if (end < pages - 1) result.push("...");
    result.push(pages);

    return result;
  }

  function buttonClass(isActive, isDisabled) {
    const base =
      "inline-flex h-9 min-w-9 items-center justify-center rounded-lg border px-3 text-xs font-black transition-colors";
    if (isDisabled) {
      return `${base} cursor-not-allowed border-slate-200 bg-slate-50 text-slate-300`;
    }
    if (isActive) {
      return `${base} border-slate-900 bg-slate-900 text-white shadow-sm`;
    }
    return `${base} border-slate-200 bg-white text-slate-600 hover:bg-slate-50 hover:text-slate-900`;
  }

  function render(target, state, items, options = {}) {
    const container = resolveContainer(target);
    if (!container) return getPage(items, state);

    const data = getPage(items, state);
    const itemLabel = options.itemLabel || state.itemLabel || DEFAULT_ITEM_LABEL;

    if (data.totalItems <= data.pageSize && !options.alwaysShow) {
      container.innerHTML = "";
      return data;
    }

    const previousDisabled = data.page <= 1;
    const nextDisabled = data.page >= data.totalPages;
    const buttons = pageList(data.page, data.totalPages)
      .map((page) => {
        if (page === "...") {
          return `<span class="inline-flex h-9 min-w-9 items-center justify-center px-2 text-xs font-black text-slate-300">...</span>`;
        }
        const isActive = page === data.page;
        return `
          <button
            type="button"
            data-page="${page}"
            class="${buttonClass(isActive, false)}"
            aria-current="${isActive ? "page" : "false"}"
          >
            ${page}
          </button>
        `;
      })
      .join("");

    container.innerHTML = `
      <div class="mt-5 flex flex-col gap-3 border-t border-slate-100 pt-4 sm:flex-row sm:items-center sm:justify-between">
        <p class="text-xs font-bold text-slate-400">
          Menampilkan ${data.startIndex}-${data.endIndex} dari ${data.totalItems} ${itemLabel}
        </p>
        <div class="flex flex-wrap items-center gap-2">
          <button
            type="button"
            data-page-action="prev"
            class="${buttonClass(false, previousDisabled)}"
            ${previousDisabled ? "disabled" : ""}
          >
            Sebelumnya
          </button>
          <div class="flex flex-wrap items-center gap-1.5">${buttons}</div>
          <button
            type="button"
            data-page-action="next"
            class="${buttonClass(false, nextDisabled)}"
            ${nextDisabled ? "disabled" : ""}
          >
            Berikutnya
          </button>
        </div>
      </div>
    `;

    container.querySelectorAll("[data-page]").forEach((button) => {
      button.addEventListener("click", () => {
        state.page = Number(button.dataset.page) || 1;
        if (typeof options.onPageChange === "function") options.onPageChange(state.page);
      });
    });

    const prevButton = container.querySelector('[data-page-action="prev"]');
    const nextButton = container.querySelector('[data-page-action="next"]');

    if (prevButton) {
      prevButton.addEventListener("click", () => {
        if (previousDisabled) return;
        state.page = Math.max(1, data.page - 1);
        if (typeof options.onPageChange === "function") options.onPageChange(state.page);
      });
    }

    if (nextButton) {
      nextButton.addEventListener("click", () => {
        if (nextDisabled) return;
        state.page = Math.min(data.totalPages, data.page + 1);
        if (typeof options.onPageChange === "function") options.onPageChange(state.page);
      });
    }

    return data;
  }

  window.SupplierHubPagination = {
    create,
    getPage,
    render,
    reset,
  };
})();
