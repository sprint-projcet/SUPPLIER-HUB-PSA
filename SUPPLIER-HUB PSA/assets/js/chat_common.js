const SupplierHubChat = (() => {
  const role = document.body.dataset.chatRole || "user";
  const refreshIntervalMs = 4000;
  const navBaseClass =
    "w-full flex items-center gap-3 px-4 py-3.5 rounded-xl transition-all duration-200 group";
  const navActiveClass =
    "bg-emerald-600 text-white shadow-xl shadow-emerald-900/40";
  const navIdleClass =
    "text-slate-400 hover:bg-slate-800 hover:text-white";

  const roleConfig = {
    user: {
      roleLabel: "UMKM",
      menuLabel: "UMKM MENU",
      pageTitle: "Chat Supplier",
      pageSubtitle: "Diskusi stok, harga, dan pengiriman langsung dengan toko supplier.",
      activeHref: "umkm_chat.html",
      redirectUrl: "../Login/login.html",
      emptyTitle: "Belum ada chat",
      emptyText: "Buka katalog dan pilih Chat Supplier dari etalase toko.",
      counterpartFallback: "Supplier",
      nav: [
        ["umkm.html", "layout-dashboard", "Dashboard"],
        ["umkm_katalog.html", "store", "Katalog Produk"],
        ["umkm_chat.html", "message-circle", "Chat"],
        ["umkm_pesanan_saya.html", "shopping-cart", "Pesanan Saya"],
        ["umkm_lacak_paket.html", "truck", "Lacak Paket"],
        ["umkm_wishlist.html", "heart", "Wishlist"],
        ["umkm_profil.html", "user-cog", "Profil UMKM"],
        ["umkm_bantuan.html", "help-circle", "Bantuan"],
      ],
      quickMessages: [
        "Halo, apakah stok produk ini tersedia?",
        "Bisa bantu info estimasi pengiriman?",
        "Apakah ada harga khusus untuk pembelian banyak?",
      ],
    },
    supplier: {
      roleLabel: "SUPPLIER",
      menuLabel: "SUPPLIER MENU",
      pageTitle: "Chat UMKM",
      pageSubtitle: "Balas pertanyaan UMKM yang masuk dari katalog supplier.",
      activeHref: "supplier_chat.html",
      redirectUrl: "../Login/login.html",
      emptyTitle: "Belum ada chat masuk",
      emptyText: "Percakapan akan muncul otomatis saat UMKM menghubungi toko Anda.",
      counterpartFallback: "UMKM",
      nav: [
        ["supplier.html", "layout-dashboard", "Dashboard"],
        ["supplier_produk_saya.html", "package", "Produk Saya"],
        ["supplier_daftar_pesanan.html", "shopping-cart", "Daftar Pesanan"],
        ["supplier_chat.html", "message-circle", "Chat UMKM"],
        ["supplier_notifikasi.html", "bell-ring", "Notifikasi"],
        ["supplier_analitik.html", "bar-chart-3", "Analitik Toko"],
        ["supplier_toko.html", "settings", "Toko Saya"],
      ],
      quickMessages: [
        "Halo, stok tersedia. Mau dibantu buat pesanan?",
        "Untuk pengiriman, estimasinya 1-3 hari kerja.",
        "Baik, saya cek detail produk dan harga terbaiknya dulu.",
      ],
    },
  };

  const state = {
    session: null,
    conversations: [],
    activeConversationID: "",
    activeConversation: null,
    messages: [],
    refreshTimer: null,
    isSending: false,
    hasLoadedConversations: false,
    conversationMarkers: {},
    lastMessageIDs: {},
  };

  function config() {
    return roleConfig[role] || roleConfig.user;
  }

  function escapeHTML(value) {
    return String(value ?? "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  function initials(value) {
    const words = String(value || "")
      .trim()
      .split(/\s+/)
      .filter(Boolean);
    if (words.length === 0) return role === "supplier" ? "U" : "S";
    return words
      .slice(0, 2)
      .map((word) => word[0])
      .join("")
      .toUpperCase();
  }

  function formatChatTime(value) {
    if (!value) return "";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "";
    return date.toLocaleTimeString("id-ID", {
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  function formatListTime(value) {
    if (!value) return "";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "";
    const today = new Date();
    const isToday = date.toDateString() === today.toDateString();
    if (isToday) return formatChatTime(value);
    return date.toLocaleDateString("id-ID", {
      day: "2-digit",
      month: "short",
    });
  }

  function notify(type, message) {
    if (typeof window.showGlobalToast === "function") {
      window.showGlobalToast(type, message);
    }
  }

  function messagePreview(value, maxLength = 72) {
    const text = String(value || "").trim();
    if (text.length <= maxLength) return text;
    return `${text.slice(0, maxLength - 3)}...`;
  }

  function conversationName(conversation) {
    return (
      conversation?.counterpart?.business_name ||
      (role === "supplier" ? "UMKM" : "Supplier")
    );
  }

  function notifyIncomingChat(conversation, messageText) {
    notify(
      "info",
      `Pesan baru dari ${conversationName(conversation)}: ${messagePreview(messageText)}`,
    );
  }

  async function apiFetch(path, options = {}) {
    const headers = {
      ...(options.headers || {}),
    };

    if (!(options.body instanceof FormData)) {
      headers["Content-Type"] = headers["Content-Type"] || "application/json";
    }

    if (state.session?.token) {
      headers.Authorization = `Bearer ${state.session.token}`;
    }

    const response = await fetch(buildApiUrl(path), {
      ...options,
      headers,
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(data.error || "Request chat gagal diproses");
    }
    return data;
  }

  function navLinkHTML(item) {
    const [href, icon, label] = item;
    const isActive = href === config().activeHref;
    return `
      <a href="${href}" class="${navBaseClass} ${isActive ? navActiveClass : navIdleClass}">
        <i data-lucide="${icon}" class="${isActive ? "text-white" : "group-hover:text-emerald-400"}"></i>
        <span class="font-semibold text-sm">${label}</span>
      </a>
    `;
  }

  function renderShell() {
    const app = document.getElementById("app-content");
    if (!app) return;

    const displayName =
      state.session.business_name ||
      state.session.name ||
      state.session.email ||
      config().roleLabel;

    app.innerHTML = `
      <aside id="sidebar" class="fixed inset-y-0 left-0 z-50 w-72 bg-slate-900 text-white transition-all duration-300 transform -translate-x-full lg:relative lg:translate-x-0 border-r border-slate-800 flex flex-col">
        <div class="p-8 flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-xl flex items-center justify-center">
              <img src="../assets/img/logo.png" alt="Logo SupplierHub" class="w-12 h-12 object-contain" />
            </div>
            <span class="text-xl font-bold tracking-tight">SUPPLIER<span class="text-emerald-400">HUB</span></span>
          </div>
          <button type="button" onclick="SupplierHubChat.toggleSidebar()" class="lg:hidden text-slate-400">
            <i data-lucide="x"></i>
          </button>
        </div>

        <nav class="mt-4 px-6 space-y-1.5 flex-1 overflow-y-auto custom-scrollbar">
          <p class="px-4 text-[10px] font-black text-slate-500 uppercase tracking-[0.2em] mb-4">${config().menuLabel}</p>
          ${config().nav.map(navLinkHTML).join("")}
        </nav>

        <div class="px-6 pb-8">
          <button onclick="logoutUser('../Login/login.html')" class="w-full flex items-center justify-between px-4 py-3 text-slate-400 hover:text-red-400 transition-colors group">
            <div class="flex items-center gap-3">
              <i data-lucide="log-out" size="20"></i>
              <span class="font-semibold text-sm">Keluar Sistem</span>
            </div>
          </button>
        </div>
      </aside>

      <main class="flex-1 flex flex-col h-screen overflow-hidden">
        <header class="h-20 bg-white border-b border-slate-200 flex items-center justify-between px-5 sm:px-8 sticky top-0 z-40">
          <div class="flex items-center gap-4 min-w-0">
            <button type="button" onclick="SupplierHubChat.toggleSidebar()" class="lg:hidden p-2 text-slate-600 hover:bg-slate-100 rounded-lg">
              <i data-lucide="menu"></i>
            </button>
            <div class="min-w-0">
              <h1 class="text-xl font-bold text-slate-900 tracking-tight">${config().pageTitle}</h1>
              <p class="text-xs text-slate-500 hidden md:block">${config().pageSubtitle}</p>
            </div>
          </div>
          <div class="flex items-center gap-4">
            <button type="button" onclick="window.DarkModeToggle?.toggle?.()" class="theme-toggle-btn hidden sm:inline-flex" title="Toggle Dark Mode">
              <i class="fas fa-moon"></i>
            </button>
            <div class="text-right hidden sm:block">
              <p class="text-sm font-bold text-slate-900 leading-none">${escapeHTML(displayName)}</p>
              <p class="text-[10px] text-emerald-600 font-black uppercase tracking-widest mt-1.5">Role: ${config().roleLabel}</p>
            </div>
            <div class="w-11 h-11 rounded-xl bg-emerald-50 border-2 border-white shadow-sm overflow-hidden ring-1 ring-emerald-100 flex items-center justify-center text-emerald-600 font-black">
              ${escapeHTML(initials(displayName))}
            </div>
          </div>
        </header>

        <section class="flex-1 overflow-hidden p-4 sm:p-6 lg:p-8">
          <div class="h-full grid grid-cols-1 lg:grid-cols-[22rem,1fr] gap-4">
            <aside class="bg-white rounded-2xl border border-slate-200 overflow-hidden flex flex-col min-h-[18rem]">
              <div class="p-4 border-b border-slate-100">
                <div class="flex items-center justify-between gap-3 mb-3">
                  <div>
                    <p class="text-xs font-black uppercase tracking-[0.2em] text-emerald-600">Pesan</p>
                    <h2 class="text-lg font-black text-slate-900">Inbox</h2>
                  </div>
                  <button type="button" id="refresh-chat-btn" class="h-10 w-10 rounded-xl border border-slate-200 text-slate-500 hover:text-emerald-600 hover:border-emerald-200 transition-all" title="Refresh">
                    <i data-lucide="refresh-cw" class="mx-auto h-4 w-4"></i>
                  </button>
                </div>
                <div class="relative">
                  <i data-lucide="search" class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400"></i>
                  <input id="chat-search" type="text" placeholder="Cari percakapan..." class="w-full rounded-xl border border-slate-200 bg-slate-50 py-3 pl-10 pr-4 text-sm outline-none focus:border-emerald-400 focus:bg-white focus:ring-4 focus:ring-emerald-50">
                </div>
              </div>
              <div id="conversation-list" class="flex-1 overflow-y-auto custom-scrollbar"></div>
            </aside>

            <section class="bg-[#f5f5f5] rounded-2xl border border-slate-200 overflow-hidden flex flex-col min-h-[34rem]">
              <div id="chat-header" class="bg-white border-b border-slate-200 px-5 py-4"></div>
              <div id="message-list" class="flex-1 overflow-y-auto custom-scrollbar px-4 sm:px-6 py-5 space-y-4"></div>
              <form id="chat-form" class="bg-white border-t border-slate-200 p-4">
                <div id="quick-message-list" class="flex flex-wrap gap-2 mb-3"></div>
                <div class="flex items-end gap-3">
                  <textarea id="chat-input" rows="1" maxlength="2000" placeholder="Tulis pesan..." class="max-h-32 flex-1 resize-none rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm outline-none focus:border-emerald-400 focus:bg-white focus:ring-4 focus:ring-emerald-50"></textarea>
                  <button id="send-chat-btn" type="submit" class="h-12 w-12 rounded-2xl bg-emerald-600 text-white shadow-lg shadow-emerald-200 transition-all hover:bg-emerald-700 disabled:cursor-not-allowed disabled:bg-slate-300 disabled:shadow-none">
                    <i data-lucide="send" class="mx-auto h-5 w-5"></i>
                  </button>
                </div>
              </form>
            </section>
          </div>
        </section>
      </main>
    `;

    app.classList.remove("invisible");
    if (window.lucide) lucide.createIcons();
    renderQuickMessages();
  }

  function renderQuickMessages() {
    const container = document.getElementById("quick-message-list");
    if (!container) return;
    container.innerHTML = config().quickMessages
      .map(
        (message) => `
          <button type="button" data-quick-message="${escapeHTML(message)}" class="rounded-full border border-emerald-100 bg-emerald-50 px-3 py-1.5 text-xs font-bold text-emerald-700 hover:border-emerald-200 hover:bg-emerald-100 transition-all">
            ${escapeHTML(message)}
          </button>
        `,
      )
      .join("");
  }

  function renderConversationList() {
    const container = document.getElementById("conversation-list");
    if (!container) return;

    const searchValue = String(document.getElementById("chat-search")?.value || "").toLowerCase();
    const visibleConversations = state.conversations.filter((conversation) => {
      const counterpart = conversation.counterpart || {};
      return [
        counterpart.business_name,
        counterpart.email,
        counterpart.region,
        conversation.last_message,
      ].some((value) => String(value || "").toLowerCase().includes(searchValue));
    });

    if (visibleConversations.length === 0) {
      container.innerHTML = `
        <div class="p-8 text-center">
          <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-500">
            <i data-lucide="message-circle"></i>
          </div>
          <p class="font-black text-slate-900">${config().emptyTitle}</p>
          <p class="mt-2 text-sm leading-6 text-slate-500">${config().emptyText}</p>
        </div>
      `;
      if (window.lucide) lucide.createIcons();
      return;
    }

    container.innerHTML = visibleConversations
      .map((conversation) => {
        const counterpart = conversation.counterpart || {};
        const name = counterpart.business_name || config().counterpartFallback;
        const isActive = conversation.id === state.activeConversationID;
        const unread = Number(conversation.unread_count || 0);
        const lastMessage = conversation.last_message || "Percakapan baru";
        const meta = counterpart.region || counterpart.category || counterpart.email || "-";
        return `
          <button type="button" data-conversation-id="${escapeHTML(conversation.id)}" class="w-full text-left px-4 py-4 border-b border-slate-100 transition-all ${isActive ? "bg-emerald-50" : "bg-white hover:bg-slate-50"}">
            <div class="flex gap-3">
              <div class="relative flex h-12 w-12 shrink-0 items-center justify-center rounded-full ${isActive ? "bg-emerald-600 text-white" : "bg-slate-100 text-slate-700"} font-black">
                ${escapeHTML(initials(name))}
                ${unread > 0 ? `<span class="absolute -right-1 -top-1 flex h-5 min-w-5 items-center justify-center rounded-full bg-emerald-500 px-1 text-[10px] font-black text-white">${unread}</span>` : ""}
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-start justify-between gap-2">
                  <p class="truncate text-sm font-black text-slate-900">${escapeHTML(name)}</p>
                  <span class="shrink-0 text-[10px] font-bold text-slate-400">${escapeHTML(formatListTime(conversation.last_message_at || conversation.updated_at))}</span>
                </div>
                <p class="mt-0.5 truncate text-[11px] font-semibold text-slate-400">${escapeHTML(meta)}</p>
                <p class="mt-1 truncate text-xs ${unread > 0 ? "font-black text-slate-800" : "font-medium text-slate-500"}">${escapeHTML(lastMessage)}</p>
              </div>
            </div>
          </button>
        `;
      })
      .join("");
  }

  function renderChatHeader() {
    const header = document.getElementById("chat-header");
    if (!header) return;

    if (!state.activeConversation) {
      header.innerHTML = `
        <div class="flex items-center gap-3">
          <div class="flex h-11 w-11 items-center justify-center rounded-full bg-slate-100 text-slate-400">
            <i data-lucide="message-circle"></i>
          </div>
          <div>
            <p class="text-sm font-black text-slate-900">Pilih percakapan</p>
            <p class="text-xs text-slate-500">Chat akan tampil di panel ini.</p>
          </div>
        </div>
      `;
      if (window.lucide) lucide.createIcons();
      return;
    }

    const counterpart = state.activeConversation.counterpart || {};
    const name = counterpart.business_name || config().counterpartFallback;
    const meta = [counterpart.category, counterpart.region, counterpart.email].filter(Boolean).join(" • ");
    header.innerHTML = `
      <div class="flex items-center justify-between gap-4">
        <div class="flex min-w-0 items-center gap-3">
          <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-emerald-600 text-white font-black">${escapeHTML(initials(name))}</div>
          <div class="min-w-0">
            <p class="truncate text-sm font-black text-slate-900">${escapeHTML(name)}</p>
            <p class="truncate text-xs text-slate-500">${escapeHTML(meta || "SupplierHub Chat")}</p>
          </div>
        </div>
        <span class="hidden sm:inline-flex items-center gap-2 rounded-full bg-emerald-50 px-3 py-1.5 text-xs font-black text-emerald-700">
          <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
          Terhubung
        </span>
      </div>
    `;
  }

  function renderMessages() {
    const container = document.getElementById("message-list");
    const form = document.getElementById("chat-form");
    if (!container || !form) return;

    form.classList.toggle("hidden", !state.activeConversation);

    if (!state.activeConversation) {
      container.innerHTML = `
        <div class="flex h-full items-center justify-center text-center">
          <div>
            <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-3xl bg-white text-emerald-500 shadow-sm">
              <i data-lucide="messages-square"></i>
            </div>
            <p class="text-sm font-black text-slate-900">${config().emptyTitle}</p>
            <p class="mt-2 max-w-xs text-sm leading-6 text-slate-500">${config().emptyText}</p>
          </div>
        </div>
      `;
      if (window.lucide) lucide.createIcons();
      return;
    }

    if (state.messages.length === 0) {
      container.innerHTML = `
        <div class="flex h-full items-center justify-center text-center">
          <div>
            <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-3xl bg-white text-emerald-500 shadow-sm">
              <i data-lucide="send"></i>
            </div>
            <p class="text-sm font-black text-slate-900">Mulai percakapan</p>
            <p class="mt-2 max-w-xs text-sm leading-6 text-slate-500">Tanyakan stok, harga, minimal order, atau jadwal pengiriman.</p>
          </div>
        </div>
      `;
      if (window.lucide) lucide.createIcons();
      return;
    }

    container.innerHTML = state.messages
      .map((message) => {
        const isMine = message.sender_id === state.session.id;
        const bubbleClass = isMine
          ? "bg-emerald-600 text-white rounded-br-md"
          : "bg-white text-slate-800 border border-slate-200 rounded-bl-md";
        const timeClass = isMine ? "text-emerald-100" : "text-slate-400";
        return `
          <div class="flex ${isMine ? "justify-end" : "justify-start"}">
            <div class="max-w-[82%] sm:max-w-[70%]">
              <div class="rounded-2xl px-4 py-3 shadow-sm ${bubbleClass}">
                <p class="whitespace-pre-wrap break-words text-sm leading-6">${escapeHTML(message.message)}</p>
                <p class="mt-2 text-right text-[10px] font-semibold ${timeClass}">
                  ${escapeHTML(formatChatTime(message.created_at))}
                  ${isMine ? `<span class="ml-1">${message.is_read ? "Dibaca" : "Terkirim"}</span>` : ""}
                </p>
              </div>
            </div>
          </div>
        `;
      })
      .join("");

    requestAnimationFrame(() => {
      container.scrollTop = container.scrollHeight;
    });
  }

  async function loadConversations(options = {}) {
    try {
      const result = await apiFetch("/api/chat/conversations");
      const nextConversations = result.data || [];
      const nextMarkers = {};

      nextConversations.forEach((conversation) => {
        const marker = String(
          conversation.last_message_at ||
            conversation.updated_at ||
            conversation.last_message ||
            "",
        );
        if (!marker) return;
        nextMarkers[conversation.id] = marker;

        const isIncoming =
          conversation.last_sender_id &&
          conversation.last_sender_id !== state.session.id &&
          Number(conversation.unread_count || 0) > 0;
        const markerChanged =
          state.conversationMarkers[conversation.id] &&
          state.conversationMarkers[conversation.id] !== marker;

        if (
          state.hasLoadedConversations &&
          isIncoming &&
          markerChanged &&
          conversation.id !== state.activeConversationID
        ) {
          notifyIncomingChat(conversation, conversation.last_message);
        }
      });

      state.conversations = nextConversations;
      state.conversationMarkers = nextMarkers;
      state.hasLoadedConversations = true;
      if (state.activeConversationID) {
        state.activeConversation =
          state.conversations.find((item) => item.id === state.activeConversationID) ||
          state.activeConversation;
      }
      renderConversationList();
      renderChatHeader();
    } catch (error) {
      if (!options.silent) notify("danger", error.message);
    }
  }

  async function selectConversation(conversationID) {
    state.activeConversationID = conversationID;
    state.activeConversation =
      state.conversations.find((item) => item.id === conversationID) || null;
    renderConversationList();
    renderChatHeader();
    await loadMessages();
  }

  async function loadMessages(options = {}) {
    if (!state.activeConversationID) {
      state.messages = [];
      renderMessages();
      return;
    }

    try {
      const result = await apiFetch(`/api/chat/conversations/${encodeURIComponent(state.activeConversationID)}/messages`);
      const nextMessages = result.data || [];
      const newestMessage = nextMessages[nextMessages.length - 1];
      const previousNewestID = state.lastMessageIDs[state.activeConversationID];

      if (
        previousNewestID &&
        newestMessage &&
        newestMessage.id !== previousNewestID &&
        newestMessage.sender_id !== state.session.id
      ) {
        notifyIncomingChat(result.conversation || state.activeConversation, newestMessage.message);
      }

      state.lastMessageIDs[state.activeConversationID] = newestMessage?.id || "";
      state.messages = nextMessages;
      if (result.conversation) state.activeConversation = result.conversation;
      renderChatHeader();
      renderMessages();
      await loadConversations({ silent: true });
    } catch (error) {
      if (!options.silent) notify("danger", error.message);
    }
  }

  async function createConversationFromSupplier(supplierID) {
    if (!supplierID || role !== "user") return "";
    const result = await apiFetch("/api/chat/conversations", {
      method: "POST",
      body: JSON.stringify({ supplier_id: supplierID }),
    });
    return result.data?.id || "";
  }

  async function sendMessage(event) {
    event.preventDefault();
    if (!state.activeConversationID || state.isSending) return;

    const input = document.getElementById("chat-input");
    const sendButton = document.getElementById("send-chat-btn");
    const message = String(input?.value || "").trim();
    if (!message) return;

    state.isSending = true;
    if (sendButton) sendButton.disabled = true;

    try {
      await apiFetch(`/api/chat/conversations/${encodeURIComponent(state.activeConversationID)}/messages`, {
        method: "POST",
        body: JSON.stringify({ message }),
      });
      input.value = "";
      input.style.height = "auto";
      await loadMessages({ silent: true });
      await loadConversations({ silent: true });
    } catch (error) {
      notify("danger", error.message);
    } finally {
      state.isSending = false;
      if (sendButton) sendButton.disabled = false;
      input?.focus();
    }
  }

  function bindEvents() {
    document.getElementById("conversation-list")?.addEventListener("click", (event) => {
      const button = event.target.closest("[data-conversation-id]");
      if (!button) return;
      selectConversation(button.dataset.conversationId);
    });

    document.getElementById("chat-search")?.addEventListener("input", renderConversationList);
    document.getElementById("refresh-chat-btn")?.addEventListener("click", async () => {
      await loadConversations();
      await loadMessages({ silent: true });
    });
    document.getElementById("chat-form")?.addEventListener("submit", sendMessage);
    document.getElementById("quick-message-list")?.addEventListener("click", (event) => {
      const button = event.target.closest("[data-quick-message]");
      if (!button) return;
      const input = document.getElementById("chat-input");
      if (!input) return;
      input.value = button.dataset.quickMessage || "";
      input.focus();
    });
    document.getElementById("chat-input")?.addEventListener("input", (event) => {
      const input = event.target;
      input.style.height = "auto";
      input.style.height = `${Math.min(input.scrollHeight, 128)}px`;
    });
    document.getElementById("chat-input")?.addEventListener("keydown", (event) => {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        document.getElementById("chat-form")?.requestSubmit();
      }
    });
  }

  async function initData() {
    const params = new URLSearchParams(window.location.search);
    let requestedConversationID = params.get("conversation_id") || "";

    try {
      const supplierID = params.get("supplier_id") || "";
      if (role === "user" && supplierID) {
        requestedConversationID = await createConversationFromSupplier(supplierID);
        if (requestedConversationID) {
          const nextUrl = `${config().activeHref}?conversation_id=${encodeURIComponent(requestedConversationID)}`;
          window.history.replaceState(null, "", nextUrl);
        }
      }

      await loadConversations({ silent: true });
      if (requestedConversationID) {
        await selectConversation(requestedConversationID);
      } else if (state.conversations.length > 0) {
        await selectConversation(state.conversations[0].id);
      } else {
        renderConversationList();
        renderChatHeader();
        renderMessages();
      }
    } catch (error) {
      notify("danger", error.message);
      renderConversationList();
      renderChatHeader();
      renderMessages();
    }
  }

  async function init() {
    state.session = checkAuth(config().redirectUrl);
    if (!state.session) return;
    if (state.session.role !== role) {
      window.location.href = config().redirectUrl;
      return;
    }

    renderShell();
    bindEvents();
    await initData();

    state.refreshTimer = setInterval(async () => {
      await loadConversations({ silent: true });
      await loadMessages({ silent: true });
    }, refreshIntervalMs);
  }

  function toggleSidebar() {
    document.getElementById("sidebar")?.classList.toggle("-translate-x-full");
  }

  window.addEventListener("beforeunload", () => {
    if (state.refreshTimer) clearInterval(state.refreshTimer);
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  return {
    toggleSidebar,
  };
})();
