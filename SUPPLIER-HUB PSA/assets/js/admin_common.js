const AdminDashboard = (() => {
  const UNREAD_NOTICE_KEY_PREFIX = "supplierhub_admin_unread_notice";

  function getSession() {
    return typeof getStoredUserSession === "function"
      ? getStoredUserSession()
      : null;
  }

  function requireAdmin(redirectUrl = "../Login/login.html") {
    const user = checkAuth(redirectUrl);
    if (!user) return null;

    if (user.role !== "admin") {
      window.location.href = redirectUrl;
      return null;
    }

    return user;
  }

  async function apiFetch(path, options = {}) {
    const session = getSession();
    const headers = {
      ...(options.headers || {}),
    };

    if (!(options.body instanceof FormData)) {
      headers["Content-Type"] = headers["Content-Type"] || "application/json";
    }
    if (session && session.token) {
      headers.Authorization = `Bearer ${session.token}`;
    }

    const response = await fetch(buildApiUrl(path), {
      ...options,
      headers,
    });
    const data = await response.json().catch(() => ({}));

    if (!response.ok) {
      const error = new Error(data.error || "Request admin gagal diproses");
      error.status = response.status;
      error.data = data;
      throw error;
    }

    return data;
  }

  function formatRupiah(value) {
    return new Intl.NumberFormat("id-ID", {
      style: "currency",
      currency: "IDR",
      minimumFractionDigits: 0,
    }).format(Number(value) || 0);
  }

  function formatNumber(value) {
    return new Intl.NumberFormat("id-ID").format(Number(value) || 0);
  }

  function formatDate(value) {
    if (!value) return "-";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "-";
    return date.toLocaleDateString("id-ID", {
      day: "numeric",
      month: "short",
      year: "numeric",
    });
  }

  function escapeHTML(value) {
    return String(value ?? "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  function shortID(prefix, value) {
    const id = String(value || "");
    if (!id) return `${prefix}-`;
    return `${prefix}-${id.slice(0, 8).toUpperCase()}`;
  }

  function orderStatusMeta(status) {
    const normalized = String(status || "").toLowerCase();
    const map = {
      pending: ["PENDING", "bg-yellow-50 text-yellow-600"],
      pending_supplier_confirmation: [
        "MENUNGGU SUPPLIER",
        "bg-orange-50 text-orange-600",
      ],
      rejected_by_supplier: ["DITOLAK", "bg-red-50 text-red-600"],
      stock_unavailable: ["STOK HABIS", "bg-red-50 text-red-600"],
      supplier_confirmed: ["STOK OK", "bg-teal-50 text-teal-600"],
      payment_pending: ["MENUNGGU BAYAR", "bg-amber-50 text-amber-600"],
      payment_request_failed: ["PAYMENT GAGAL", "bg-red-50 text-red-600"],
      paid: ["LUNAS", "bg-emerald-50 text-emerald-600"],
      payment_failed: ["GAGAL BAYAR", "bg-red-50 text-red-600"],
      shipment_created: ["PENGIRIMAN", "bg-indigo-50 text-indigo-600"],
      processing: ["PROSES", "bg-blue-50 text-blue-600"],
      shipped: ["DIKIRIM", "bg-indigo-50 text-indigo-600"],
      completed: ["SELESAI", "bg-emerald-50 text-emerald-600"],
      cancelled: ["BATAL", "bg-red-50 text-red-600"],
    };

    return map[normalized] || [
      normalized ? normalized.toUpperCase() : "-",
      "bg-slate-100 text-slate-600",
    ];
  }

  function supplierStatusMeta(status) {
    const normalized = String(status || "").toLowerCase();
    const map = {
      active: ["AKTIF", "bg-emerald-50 text-emerald-600"],
      pending: ["MENUNGGU VERIFIKASI", "bg-slate-100 text-slate-600"],
      suspended: ["DITANGGUHKAN", "bg-red-50 text-red-600"],
    };

    return map[normalized] || [
      normalized ? normalized.toUpperCase() : "-",
      "bg-slate-100 text-slate-600",
    ];
  }

  function badge(label, className) {
    return `<span class="text-[10px] font-black px-3 py-1 rounded-full ${className}">${escapeHTML(label)}</span>`;
  }

  function formatNotificationTime(value) {
    if (!value) return "-";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "-";
    return date.toLocaleString("id-ID", {
      day: "2-digit",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function notify(type, message) {
    if (typeof window.showGlobalToast === "function") {
      window.showGlobalToast(type, message);
      return;
    }
    if (typeof window.showToast === "function") {
      window.showToast(type, message);
      return;
    }
    alert(message);
  }

  function getNotifications(options = {}) {
    const query = options.unreadOnly ? "?unread_only=true" : "";
    return apiFetch(`/api/admin/notifications${query}`).then(
      (result) => result.data || [],
    );
  }

  function markNotificationRead(notificationID) {
    return apiFetch(`/api/admin/notifications/${encodeURIComponent(notificationID)}/read`, {
      method: "PUT",
      body: JSON.stringify({}),
    });
  }

  function unreadNoticeKey() {
    const session = getSession() || {};
    return `${UNREAD_NOTICE_KEY_PREFIX}_${session.id || session.email || "guest"}`;
  }

  function unreadNoticeSignature(notifications) {
    return `${notifications.length}:${notifications
      .slice(0, 5)
      .map((notification) =>
        [
          notification.id,
          notification.source_id,
          notification.created_at,
          notification.title,
        ]
          .filter(Boolean)
          .join(":"),
      )
      .join("|")}`;
  }

  function shouldShowUnreadNotice(notifications) {
    if (!Array.isArray(notifications) || notifications.length === 0) return false;

    const signature = unreadNoticeSignature(notifications);
    try {
      const key = unreadNoticeKey();
      if (sessionStorage.getItem(key) === signature) return false;
      sessionStorage.setItem(key, signature);
    } catch (error) {
      return true;
    }

    return true;
  }

  function renderNotificationPanel(notifications) {
    document.getElementById("admin-notification-panel")?.remove();
    if (!Array.isArray(notifications) || notifications.length === 0) return;

    const primary = notifications[0];
    const remainingCount = Math.max(0, notifications.length - 1);
    const title =
      notifications.length > 1
        ? `${notifications.length} notifikasi admin belum dibaca`
        : primary.title || "Notifikasi Admin";
    const message =
      notifications.length > 1
        ? `Terbaru: ${primary.message || "-"}`
        : primary.message || "-";
    const createdAt = formatNotificationTime(primary.created_at);

    const panel = document.createElement("div");
    panel.id = "admin-notification-panel";
    panel.className =
      "fixed left-1/2 top-24 z-[120] w-[calc(100%-2rem)] max-w-sm -translate-x-1/2";
    panel.innerHTML = `
      <div class="notification-toast animate-slide-in overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl shadow-slate-900/10 backdrop-blur-md">
        <button type="button" class="flex w-full items-start gap-3 px-4 py-3 text-left" data-action="open">
          <div class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-emerald-50 text-emerald-600">
            <i data-lucide="check-circle" class="h-4 w-4"></i>
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-start justify-between gap-3">
              <p class="text-sm font-bold leading-5 text-slate-900">${escapeHTML(title)}</p>
              ${
                remainingCount > 0
                  ? `<span class="shrink-0 rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-black text-slate-500">+${remainingCount}</span>`
                  : ""
              }
            </div>
            <p class="mt-1 line-clamp-2 text-xs leading-5 text-slate-500">${escapeHTML(message)}</p>
            <p class="mt-2 text-[10px] font-bold uppercase tracking-widest text-slate-400">${escapeHTML(createdAt)}</p>
          </div>
        </button>
        <div class="flex items-center justify-between gap-2 border-t border-slate-100 px-4 py-2">
          <span class="text-[10px] font-black uppercase tracking-widest text-slate-400">${notifications.length > 1 ? "Ringkasan admin" : "Notifikasi baru"}</span>
          <div class="flex items-center gap-2">
            <button type="button" data-action="close" class="rounded-lg px-2 py-1 text-xs font-bold text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700">Tutup</button>
            <button type="button" data-action="open" class="rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-bold text-white transition-colors hover:bg-emerald-700">Buka</button>
          </div>
        </div>
      </div>
    `;

    const card = panel.querySelector(".notification-toast");
    let autoHideTimer = null;
    const dismissPanel = () => {
      if (card.dataset.removing === "true") return;
      card.dataset.removing = "true";
      clearTimeout(autoHideTimer);
      card.classList.add("animate-slide-out");
      setTimeout(() => panel.remove(), 300);
    };
    const openPrimaryNotification = () => {
      const query =
        notifications.length === 1 && primary.source_id
          ? `?q=${encodeURIComponent(primary.source_id)}`
          : "";
      window.location.href = `admin_kontrol_stok.html${query}`;
    };

    panel.querySelectorAll('[data-action="open"]').forEach((button) => {
      button.addEventListener("click", openPrimaryNotification);
    });
    panel.querySelector('[data-action="close"]')?.addEventListener("click", (event) => {
      event.stopPropagation();
      dismissPanel();
    });

    autoHideTimer = setTimeout(dismissPanel, 5500);
    card.addEventListener("mouseenter", () => clearTimeout(autoHideTimer));

    document.body.appendChild(panel);
    if (window.lucide) lucide.createIcons();
  }

  async function showUnreadNotifications() {
    try {
      const notifications = await getNotifications({ unreadOnly: true });
      if (shouldShowUnreadNotice(notifications)) {
        renderNotificationPanel(notifications);
      }
    } catch (error) {
      console.warn("Gagal mengambil notifikasi admin", error);
    }
  }

  function downloadCSV(filename, header, rows) {
    const csvRows = [
      header,
      ...rows.map((row) =>
        row
          .map((value) => `"${String(value ?? "").replace(/"/g, '""')}"`)
          .join(","),
      ),
    ];

    const blob = new Blob([csvRows.join("\n")], {
      type: "text/csv;charset=utf-8;",
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  }

  async function initPage(options = {}) {
    const user = requireAdmin(options.redirectUrl);
    if (!user) return null;

    const displayName = user.business_name || user.name || user.email || "Admin";
    document.querySelectorAll("#user-display-name").forEach((element) => {
      element.textContent = displayName;
    });

    const dateEl = document.getElementById("current-date");
    if (dateEl) {
      dateEl.textContent = new Date().toLocaleDateString("id-ID", {
        day: "numeric",
        month: "long",
        year: "numeric",
      });
    }

    const content = document.getElementById("app-content");
    if (content) content.classList.remove("invisible");

    if (window.lucide) lucide.createIcons();

    if (typeof options.onReady === "function") {
      await options.onReady(user);
    }

    await showUnreadNotifications();

    return user;
  }

  window.toggleSidebar = function toggleSidebar() {
    const sidebar = document.getElementById("sidebar");
    if (sidebar) sidebar.classList.toggle("-translate-x-full");
  };

  return {
    apiFetch,
    badge,
    downloadCSV,
    escapeHTML,
    formatDate,
    formatNumber,
    formatRupiah,
    getNotifications,
    initPage,
    markNotificationRead,
    notify,
    orderStatusMeta,
    renderNotificationPanel,
    showUnreadNotifications,
    shortID,
    supplierStatusMeta,
  };
})();
