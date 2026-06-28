// auth.js
// Autentikasi dan Konfigurasi Dashboard Multi-Role

const API_BASE_URL =
  window.SUPPLIER_HUB_API_BASE_URL || "http://localhost:8080";
const LEGACY_AUTH_KEYS = ["authToken", "userRole", "userData", "token"];

function buildApiUrl(path) {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return `${API_BASE_URL}${normalizedPath}`;
}

const SUPPLIER_HUB_TOAST_DURATION = 2500;
const SUPPLIER_HUB_TOAST_QUEUE_KEY = "supplierhub_pending_toast";
let activeToastState = null;

const toastMessages = {
  loginSuccess: "Selamat Datang! Anda berhasil masuk ke dashboard admin.",
  logoutSuccess: "Sesi Berakhir. Anda telah berhasil keluar dari sistem.",
  stockSaved: "Berhasil! Data stok bahan baku telah diperbarui di database.",
  dataDeleted: "Data Terhapus. Bahan baku berhasil dihapus dari katalog.",
  actionFailed:
    "Aksi Gagal! Mohon periksa kembali input Anda atau hak akses akun.",
};

const toastConfig = {
  success: {
    accent: "border-emerald-200/80 text-emerald-700",
    iconBg: "bg-emerald-500/10",
    progress: "bg-emerald-500",
    title: "Berhasil",
    icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.704 5.29a1 1 0 010 1.42l-7.25 7.25a1 1 0 01-1.42 0L3.29 9.21a1 1 0 111.42-1.42l4.03 4.04 6.54-6.54a1 1 0 011.42 0z" clip-rule="evenodd"/></svg>`,
  },
  info: {
    accent: "border-sky-200/80 text-sky-700",
    iconBg: "bg-sky-500/10",
    progress: "bg-sky-500",
    title: "Informasi",
    icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M18 10A8 8 0 112 10a8 8 0 0116 0zM9 9a1 1 0 012 0v5a1 1 0 11-2 0V9zm1-4a1.25 1.25 0 100 2.5A1.25 1.25 0 0010 5z" clip-rule="evenodd"/></svg>`,
  },
  warning: {
    accent: "border-amber-200/80 text-amber-700",
    iconBg: "bg-amber-500/10",
    progress: "bg-amber-500",
    title: "Perhatian",
    icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.72-1.36 3.486 0l6.516 11.584c.75 1.334-.213 2.983-1.742 2.983H3.483c-1.529 0-2.492-1.649-1.742-2.983L8.257 3.1zM11 14a1 1 0 10-2 0 1 1 0 002 0zm-1-2a1 1 0 01-1-1V8a1 1 0 112 0v3a1 1 0 01-1 1z" clip-rule="evenodd"/></svg>`,
  },
  danger: {
    accent: "border-rose-200/80 text-rose-700",
    iconBg: "bg-rose-500/10",
    progress: "bg-rose-500",
    title: "Gagal",
    icon: `<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM7.28 7.28a1 1 0 011.42 0L10 8.59l1.3-1.31a1 1 0 111.42 1.42L11.41 10l1.31 1.3a1 1 0 01-1.42 1.42L10 11.41l-1.3 1.31a1 1 0 01-1.42-1.42L8.59 10 7.28 8.7a1 1 0 010-1.42z" clip-rule="evenodd"/></svg>`,
  },
};

function ensureToastContainer() {
  let container = document.getElementById("toast-container");
  if (container) return container;

  // Satu wadah toast global agar semua halaman punya posisi notifikasi yang konsisten.
  container = document.createElement("div");
  container.id = "toast-container";
  container.className =
    "fixed left-1/2 top-4 z-[9999] flex w-[calc(100%-2rem)] max-w-sm -translate-x-1/2 flex-col items-stretch pointer-events-none sm:top-6";
  document.body.appendChild(container);
  return container;
}

function normalizeToast(type, message) {
  if (!message) {
    return { type: "info", message: String(type || "") };
  }

  return {
    type: toastConfig[type] ? type : "info",
    message: String(message),
  };
}

function removeToast(toast) {
  if (!toast || toast.dataset.removing === "true") return;
  if (activeToastState?.toast === toast) {
    clearTimeout(activeToastState.dismissTimer);
    clearTimeout(activeToastState.outsideListenerTimer);
    document.removeEventListener(
      "pointerdown",
      activeToastState.handleOutsideClick,
      true,
    );
    activeToastState = null;
  }
  toast.dataset.removing = "true";
  toast.classList.add("translate-y-3", "opacity-0", "scale-95");
  toast.classList.remove("translate-y-0", "opacity-100", "scale-100");
  setTimeout(() => toast.remove(), 320);
}

function clearActiveToast(options = {}) {
  if (!activeToastState) return;

  clearTimeout(activeToastState.dismissTimer);
  clearTimeout(activeToastState.outsideListenerTimer);
  document.removeEventListener(
    "pointerdown",
    activeToastState.handleOutsideClick,
    true,
  );

  if (options.removeNode) {
    activeToastState.toast.remove();
  }
  activeToastState = null;
}

function showToast(type, message) {
  const toastData = normalizeToast(type, message);
  const config = toastConfig[toastData.type] || toastConfig.info;
  const container = ensureToastContainer();
  clearActiveToast({ removeNode: true });
  container.replaceChildren();

  // Kartu toast tunggal. Panggilan baru menggantikan toast lama supaya pop-up tidak menumpuk.
  const toast = document.createElement("div");
  toast.dataset.supplierHubToast = "true";
  toast.setAttribute("role", toastData.type === "danger" ? "alert" : "status");
  toast.setAttribute("aria-live", toastData.type === "danger" ? "assertive" : "polite");
  toast.className = [
    "pointer-events-auto relative overflow-hidden rounded-xl border",
    "bg-white/80 p-4 pr-10 shadow-xl shadow-slate-900/10 backdrop-blur-md",
    "transition-all duration-300 ease-out translate-y-3 opacity-0 scale-95",
    config.accent,
  ].join(" ");

  const content = document.createElement("div");
  content.className = "flex items-start gap-3";

  const icon = document.createElement("div");
  icon.className = [
    "mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-lg",
    config.iconBg,
  ].join(" ");
  icon.innerHTML = config.icon;

  const text = document.createElement("div");
  text.className = "min-w-0 flex-1";

  const title = document.createElement("p");
  title.className = "text-sm font-bold leading-5 text-slate-900";
  title.textContent = config.title;

  const body = document.createElement("p");
  body.className = "mt-0.5 text-sm leading-5 text-slate-600";
  body.textContent = toastData.message;

  const closeButton = document.createElement("button");
  closeButton.type = "button";
  closeButton.className =
    "absolute right-3 top-3 rounded-md p-1 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700";
  closeButton.setAttribute("aria-label", "Tutup notifikasi");
  closeButton.innerHTML =
    '<svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor"><path d="M6.28 5.22a.75.75 0 00-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 101.06 1.06L10 11.06l3.72 3.72a.75.75 0 101.06-1.06L11.06 10l3.72-3.72a.75.75 0 00-1.06-1.06L10 8.94 6.28 5.22z"/></svg>';

  const progress = document.createElement("div");
  progress.className = [
    "absolute bottom-0 left-0 h-1 w-full rounded-b-xl",
    "transition-[width] ease-linear",
    config.progress,
  ].join(" ");
  progress.style.transitionDuration = `${SUPPLIER_HUB_TOAST_DURATION}ms`;

  text.append(title, body);
  content.append(icon, text);
  toast.append(content, closeButton, progress);
  container.appendChild(toast);

  const toastState = {
    toast,
    dismissTimer: null,
    outsideListenerTimer: null,
    handleOutsideClick: null,
  };
  activeToastState = toastState;

  // Animasi masuk: toast turun halus dari atas lalu menjadi solid.
  requestAnimationFrame(() => {
    toast.classList.remove("translate-y-3", "opacity-0", "scale-95");
    toast.classList.add("translate-y-0", "opacity-100", "scale-100");
    progress.style.width = "0%";
  });

  let dismissTimer;
  const dismiss = () => {
    removeToast(toast);
  };
  const handleOutsideClick = (event) => {
    if (!toast.contains(event.target)) dismiss();
  };

  toastState.handleOutsideClick = handleOutsideClick;
  dismissTimer = setTimeout(dismiss, SUPPLIER_HUB_TOAST_DURATION);
  toastState.dismissTimer = dismissTimer;
  closeButton.addEventListener("click", (event) => {
    event.stopPropagation();
    dismiss();
  });
  toast.addEventListener("pointerdown", (event) => event.stopPropagation());
  toastState.outsideListenerTimer = setTimeout(() => {
    if (activeToastState === toastState) {
      document.addEventListener("pointerdown", handleOutsideClick, true);
    }
  }, 0);

  return toast;
}

function queueToast(type, message) {
  sessionStorage.setItem(
    SUPPLIER_HUB_TOAST_QUEUE_KEY,
    JSON.stringify({ type, message }),
  );
}

function consumeQueuedToast() {
  const queuedToast = sessionStorage.getItem(SUPPLIER_HUB_TOAST_QUEUE_KEY);
  if (!queuedToast) return;

  sessionStorage.removeItem(SUPPLIER_HUB_TOAST_QUEUE_KEY);
  try {
    const { type, message } = JSON.parse(queuedToast);
    showToast(type, message);
  } catch (error) {
    showToast("info", queuedToast);
  }
}

window.showToast = showToast;
window.showGlobalToast = showToast;
window.queueToast = queueToast;
window.toastMessages = toastMessages;

function ensureUmkmChatNavigation() {
  const nav = document.querySelector("#sidebar nav");
  if (!nav || nav.querySelector('a[href="umkm_chat.html"]')) return;

  const link = document.createElement("a");
  link.href = "umkm_chat.html";
  link.className =
    "w-full flex items-center gap-3 px-4 py-3.5 rounded-xl transition-all duration-200 group text-slate-400 hover:bg-slate-800 hover:text-white";
  link.innerHTML = `
    <i data-lucide="message-circle" class="group-hover:text-emerald-400"></i>
    <span class="font-semibold text-sm">Chat</span>
  `;

  const beforeLink =
    nav.querySelector('a[href="umkm_pesanan_saya.html"]') ||
    nav.querySelector('a[href="umkm_bantuan.html"]');
  nav.insertBefore(link, beforeLink || null);

  if (window.lucide) lucide.createIcons();
}

const SUPPLIER_HUB_CHAT_POLL_MS = 8000;
let supplierHubChatPollTimer = null;
let supplierHubChatPollInitialized = false;

function chatPageForRole(role) {
  if (role === "supplier") return "supplier_chat.html";
  if (role === "user") return "umkm_chat.html";
  return "";
}

function chatSnapshotKey(session) {
  return `supplierhub_chat_snapshot_${session.role}_${session.id || session.email || "guest"}`;
}

function chatInitialNoticeKey(session) {
  return `supplierhub_chat_initial_notice_${session.role}_${session.id || session.email || "guest"}`;
}

function loadChatSnapshot(session) {
  try {
    return JSON.parse(localStorage.getItem(chatSnapshotKey(session)) || "{}");
  } catch (error) {
    return {};
  }
}

function saveChatSnapshot(session, snapshot) {
  localStorage.setItem(chatSnapshotKey(session), JSON.stringify(snapshot || {}));
}

function shortChatPreview(value, maxLength = 72) {
  const text = String(value || "").trim();
  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength - 3)}...`;
}

function updateChatNavigationBadge(role, totalUnread) {
  const page = chatPageForRole(role);
  if (!page) return;

  document.querySelectorAll(`a[href="${page}"]`).forEach((link) => {
    let badge = link.querySelector("[data-chat-unread-badge]");
    if (totalUnread <= 0) {
      badge?.remove();
      return;
    }

    if (!badge) {
      badge = document.createElement("span");
      badge.dataset.chatUnreadBadge = "true";
      badge.className =
        "ml-auto inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-emerald-500 px-1.5 text-[10px] font-black text-white shadow-sm";
      link.appendChild(badge);
    }

    badge.textContent = totalUnread > 99 ? "99+" : String(totalUnread);
  });
}

async function fetchChatConversationsForSession(session) {
  if (!session || !session.token) return [];

  const response = await fetch(buildApiUrl("/api/chat/conversations"), {
    headers: {
      Authorization: `Bearer ${session.token}`,
      Accept: "application/json",
    },
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.error || "Gagal mengambil notifikasi chat");
  return data.data || [];
}

function handleChatNotificationPoll(session, conversations) {
  const previousSnapshot = loadChatSnapshot(session);
  const nextSnapshot = {};
  const totalUnread = conversations.reduce(
    (sum, conversation) => sum + Number(conversation.unread_count || 0),
    0,
  );

  updateChatNavigationBadge(session.role, totalUnread);

  conversations.forEach((conversation) => {
    const marker = String(
      conversation.last_message_at ||
        conversation.updated_at ||
        conversation.last_message ||
        "",
    );
    if (!marker) return;

    nextSnapshot[conversation.id] = marker;
    const isIncoming =
      conversation.last_sender_id &&
      conversation.last_sender_id !== session.id &&
      Number(conversation.unread_count || 0) > 0;
    const isNew = previousSnapshot[conversation.id] && previousSnapshot[conversation.id] !== marker;

    if (supplierHubChatPollInitialized && isIncoming && isNew) {
      const name =
        conversation.counterpart?.business_name ||
        (session.role === "supplier" ? "UMKM" : "Supplier");
      showToast(
        "info",
        `Pesan baru dari ${name}: ${shortChatPreview(conversation.last_message)}`,
      );
    }
  });

  const initialKey = chatInitialNoticeKey(session);
  if (!supplierHubChatPollInitialized && totalUnread > 0 && !sessionStorage.getItem(initialKey)) {
    showToast("info", `Ada ${totalUnread} chat belum dibaca.`);
    sessionStorage.setItem(initialKey, "true");
  }

  supplierHubChatPollInitialized = true;
  saveChatSnapshot(session, nextSnapshot);
}

function startChatNotificationPolling(session) {
  if (!session || !["user", "supplier"].includes(session.role)) return;
  if (document.body?.dataset?.chatRole) return;
  if (supplierHubChatPollTimer) return;

  const poll = async () => {
    try {
      const latestSession = getStoredUserSession();
      if (!latestSession || latestSession.role !== session.role) return;
      const conversations = await fetchChatConversationsForSession(latestSession);
      handleChatNotificationPoll(latestSession, conversations);
    } catch (error) {
      console.warn("Gagal mengambil notifikasi chat", error);
    }
  };

  poll();
  supplierHubChatPollTimer = setInterval(poll, SUPPLIER_HUB_CHAT_POLL_MS);
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", consumeQueuedToast);
} else {
  consumeQueuedToast();
}

const roleConfig = {
  admin: {
    title: "Admin Command Center",
    stats: [
      {
        label: "Total Supplier",
        value: "128",
        icon: "users",
        change: "+12%",
        positive: true,
      },
      {
        label: "Total Transaksi",
        value: "Rp 2.4M",
        icon: "shopping-cart",
        change: "+18%",
        positive: true,
      },
      {
        label: "Pesanan Aktif",
        value: "45",
        icon: "package",
        change: "-4%",
        positive: false,
      },
      {
        label: "Revenue Growth",
        value: "24%",
        icon: "bar-chart-3",
        change: "+2%",
        positive: true,
      },
    ],
    nav: [
      { name: "Overview", icon: "layout-dashboard" },
      { name: "Daftar Supplier", icon: "users" },
      { name: "Kontrol Stok", icon: "package" },
      { name: "Keuangan", icon: "bar-chart-3" },
      { name: "Pengaturan", icon: "settings" },
    ],
    tableTitle: "Monitoring Transaksi Global",
    quickBtn: "Supplier",
  },
  supplier: {
    title: "Supplier Portal",
    stats: [
      {
        label: "Stok Barang",
        value: "1,240",
        icon: "package",
        change: "+5%",
        positive: true,
      },
      {
        label: "Pesanan Baru",
        value: "12",
        icon: "bell",
        change: "Baru",
        positive: true,
      },
      {
        label: "Pendapatan",
        value: "Rp 450jt",
        icon: "bar-chart-3",
        change: "+10%",
        positive: true,
      },
      {
        label: "Rating Toko",
        value: "4.8/5",
        icon: "check-circle-2",
        change: "Stabil",
        positive: true,
      },
    ],
    nav: [
      { name: "Dashboard", icon: "layout-dashboard" },
      { name: "Produk Saya", icon: "package" },
      { name: "Daftar Pesanan", icon: "shopping-cart" },
      { name: "Chat UMKM", icon: "message-circle" },
      { name: "Analitik Toko", icon: "bar-chart-3" },
      { name: "Toko Saya", icon: "settings" },
    ],
    tableTitle: "Pesanan Masuk Terbaru",
    quickBtn: "Produk",
  },
  user: {
    title: "Dashboard UMKM (Pembeli)",
    stats: [
      {
        label: "Total Pesanan",
        value: "24",
        icon: "shopping-cart",
        change: "Bulan ini",
        positive: true,
      },
      {
        label: "Sedang Dikirim",
        value: "3",
        icon: "truck",
        change: "Aktif",
        positive: true,
      },
      {
        label: "Voucher",
        value: "5",
        icon: "tag",
        change: "Tersedia",
        positive: true,
      },
      {
        label: "Poin Hub",
        value: "12,500",
        icon: "zap",
        change: "+500",
        positive: true,
      },
    ],
    nav: [
      { name: "Belanja", icon: "layout-dashboard" },
      { name: "Chat", icon: "message-circle" },
      { name: "Pesanan Saya", icon: "shopping-cart" },
      { name: "Lacak Paket", icon: "truck" },
      { name: "Wishlist", icon: "heart" },
      { name: "Bantuan", icon: "help-circle" },
    ],
    tableTitle: "Riwayat Pembelian Saya",
    quickBtn: "Order",
  },
};

/**
 * Menormalkan response auth backend menjadi format sesi frontend.
 */
function createUserSession(data, fallbackRole = "user") {
  const user = data.user || {};
  const role = data.role || fallbackRole;

  return {
    name: user.business_name || user.email || role,
    business_name: user.business_name || "",
    role: role,
    email: user.email || "",
    token: data.token,
    id: user.id || "",
    address: user.address || "",
    category: user.category || "",
    region: user.region || "",
    pic_name: user.pic_name || "",
    phone: user.phone || "",
    status: user.status || "",
    lastLogin: new Date().toISOString(),
  };
}

/**
 * Menyimpan sesi di satu sumber data agar dashboard membaca format yang konsisten.
 */
function saveUserSession(userSession) {
  localStorage.setItem("user_session", JSON.stringify(userSession));
  LEGACY_AUTH_KEYS.forEach((key) => localStorage.removeItem(key));
  return userSession;
}

function saveUserSessionFromAuthResponse(data, fallbackRole = "user") {
  return saveUserSession(createUserSession(data, fallbackRole));
}

function clearUserSession() {
  localStorage.removeItem("user_session");
  LEGACY_AUTH_KEYS.forEach((key) => localStorage.removeItem(key));
}

function getStoredUserSession() {
  const sessionData = localStorage.getItem("user_session");
  if (!sessionData || sessionData === "undefined" || sessionData === "null") {
    return null;
  }

  try {
    return JSON.parse(sessionData);
  } catch (e) {
    console.error("Data sesi korup:", e);
    clearUserSession();
    return null;
  }
}

/**
 * Fungsi untuk login ke sistem.
 */
function loginUser(email, password, role) {
  return fetch(buildApiUrl("/api/auth/login"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ email: email, password: password }),
  })
    .then((response) =>
      response
        .json()
        .then((data) => ({ status: response.status, ok: response.ok, data })),
    )
    .then(({ status, ok, data }) => {
      if (!ok) {
        throw new Error(data.error || "Terjadi kesalahan saat login.");
      }

      const userSession = saveUserSessionFromAuthResponse(data, role);
      const dashboardName =
        userSession.role === "admin"
          ? "admin"
          : userSession.role === "supplier"
            ? "supplier"
            : "UMKM";

      queueToast(
        "success",
        userSession.role === "admin"
          ? toastMessages.loginSuccess
          : `Selamat Datang! Anda berhasil masuk ke dashboard ${dashboardName}.`,
      );

      return userSession;
    });
}

/**
 * Memastikan sesi pengguna ada. Jika tidak, redirect ke halaman login.
 */
function checkAuth(redirectUrl = null) {
  const user = getStoredUserSession();
  if (!user) {
    if (redirectUrl) window.location.href = redirectUrl;
    return null;
  }

  return user;
}

/**
 * Fungsi Logout untuk memutus sesi.
 */
function logoutUser(redirectUrl = "../Login/login.html") {
  clearUserSession();
  queueToast("info", toastMessages.logoutSuccess);
  window.location.href = redirectUrl;
}

/**
 * Mengambil konfigurasi antarmuka/data untuk tiap role dashboard
 */
function getRoleConfig(role) {
  return roleConfig[role] || roleConfig["user"];
}

// Global dynamic branding listener for UMKM and Admin dashboards
document.addEventListener("DOMContentLoaded", () => {
  const user = getStoredUserSession();
  if (user) {
    const displayName = user.business_name || user.name || user.email || (user.role === "admin" ? "Admin" : "UMKM");
    if (user.role === "user") {
      ensureUmkmChatNavigation();
      document.querySelectorAll("[data-umkm-name-display]").forEach((element) => {
        element.textContent = displayName;
      });
      startChatNotificationPolling(user);
    } else if (user.role === "supplier") {
      startChatNotificationPolling(user);
    } else if (user.role === "admin") {
      document.querySelectorAll("[data-admin-name-display]").forEach((element) => {
        element.textContent = displayName;
      });
    }
  }
});

function showConfirm(message) {
  return new Promise((resolve) => {
    const overlay = document.createElement("div");
    overlay.className = "fixed inset-0 z-[10000] bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4 transition-all duration-300 opacity-0 scale-95";
    overlay.innerHTML = `
      <div class="bg-white rounded-3xl w-full max-w-sm p-6 shadow-2xl border border-slate-100 flex flex-col items-center text-center">
        <div class="w-12 h-12 bg-amber-50 rounded-2xl flex items-center justify-center text-amber-500 mb-4 border border-amber-100">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h3 class="text-lg font-black text-slate-900 mb-2">Konfirmasi</h3>
        <p class="text-sm text-slate-500 mb-6 leading-relaxed">${message}</p>
        <div class="flex w-full gap-3">
          <button id="confirm-btn-cancel" type="button" class="flex-1 py-3 border border-slate-200 text-slate-600 rounded-xl text-sm font-bold hover:bg-slate-50 transition-colors">Batal</button>
          <button id="confirm-btn-ok" type="button" class="flex-1 py-3 bg-emerald-600 text-white rounded-xl text-sm font-bold hover:bg-emerald-700 transition-colors shadow-lg shadow-emerald-600/20">Ya, Setuju</button>
        </div>
      </div>
    `;

    document.body.appendChild(overlay);

    requestAnimationFrame(() => {
      overlay.classList.remove("opacity-0", "scale-95");
      overlay.classList.add("opacity-100", "scale-100");
    });

    const cleanup = (value) => {
      overlay.classList.remove("opacity-100", "scale-100");
      overlay.classList.add("opacity-0", "scale-95");
      setTimeout(() => {
        overlay.remove();
        resolve(value);
      }, 300);
    };

    overlay.querySelector("#confirm-btn-cancel").addEventListener("click", () => cleanup(false));
    overlay.querySelector("#confirm-btn-ok").addEventListener("click", () => cleanup(true));
  });
}

function showAlert(message) {
  return new Promise((resolve) => {
    const overlay = document.createElement("div");
    overlay.className = "fixed inset-0 z-[10000] bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4 transition-all duration-300 opacity-0 scale-95";
    overlay.innerHTML = `
      <div class="bg-white rounded-3xl w-full max-w-sm p-6 shadow-2xl border border-slate-100 flex flex-col items-center text-center">
        <div class="w-12 h-12 bg-emerald-50 rounded-2xl flex items-center justify-center text-emerald-600 mb-4 border border-emerald-100">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <h3 class="text-lg font-black text-slate-900 mb-2">Informasi</h3>
        <p class="text-sm text-slate-500 mb-6 leading-relaxed">${message.replace(/\n/g, '<br>')}</p>
        <button id="alert-btn-ok" type="button" class="w-full py-3 bg-slate-900 text-white rounded-xl text-sm font-bold hover:bg-slate-800 transition-colors shadow-lg">OK</button>
      </div>
    `;
    document.body.appendChild(overlay);

    requestAnimationFrame(() => {
      overlay.classList.remove("opacity-0", "scale-95");
      overlay.classList.add("opacity-100", "scale-100");
    });

    const cleanup = () => {
      overlay.classList.remove("opacity-100", "scale-100");
      overlay.classList.add("opacity-0", "scale-95");
      setTimeout(() => {
        overlay.remove();
        resolve();
      }, 300);
    };

    overlay.querySelector("#alert-btn-ok").addEventListener("click", cleanup);
  });
}

window.showConfirm = showConfirm;
window.showAlert = showAlert;

