const app = document.querySelector("#app");
const toast = document.querySelector("#toast");

const PRODUCT_STATUS = {
  1: "在售",
  2: "预售中",
  3: "正在下架",
  4: "已下架",
};

const ORDER_STATUS = {
  1: "已下单未发货",
  2: "配送中",
  3: "已收货",
  4: "已退款",
};

const state = {
  token: localStorage.getItem("token") || "",
  role: localStorage.getItem("role") || "BUYER",
  apiBase: localStorage.getItem("apiBase") || "",
  theme: localStorage.getItem("theme") || "LIGHT",
  authMode: "login",
  buyerTab: "products",
  sellerTab: "products",
  buyerProfile: null,
  sellerProfile: null,
  buyerProductName: "",
  buyerOrderStatus: "",
  buyerProducts: [],
  buyerOrders: [],
  sellerProducts: [],
  sellerOrders: [],
  sellerOrderProductId: "",
  sellerOrderProductNamePrefix: "",
  trendPoints: [],
  trendHoverIndex: -1,
  trendStartDate: "",
  trendEndDate: "",
  loading: false,
};

document.body.classList.toggle("dark", state.theme === "DARK");

function money(cent) {
  return `¥${((Number(cent) || 0) / 100).toFixed(2)}`;
}

function idempotencyKey(prefix) {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function htmlEscape(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function formData(form) {
  return Object.fromEntries(new FormData(form).entries());
}

function showToast(message, type = "info") {
  toast.textContent = message;
  toast.className = `toast show ${type}`;
  window.clearTimeout(showToast.timer);
  showToast.timer = window.setTimeout(() => {
    toast.className = "toast";
  }, 2600);
}

async function apiFetch(path, options = {}) {
  const requestId = options.requestId || `web-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const headers = {
    "Content-Type": "application/json",
    "X-Request-Id": requestId,
    ...(options.headers || {}),
  };
  if (state.token) {
    headers.Authorization = `Bearer ${state.token}`;
  }
  if (options.idempotent) {
    headers["Idempotency-Key"] = idempotencyKey(options.idempotent);
  }
  const res = await fetch(`${state.apiBase}${path}`, {
    ...options,
    headers,
  });
  const json = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(`${json.message || json.code || `HTTP ${res.status}`}，requestId=${json.requestId || requestId}`);
  }
  return json.data;
}

function qs(params) {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== "") {
      query.set(key, value);
    }
  });
  const text = query.toString();
  return text ? `?${text}` : "";
}

function setSession(output, role) {
  state.token = output.token;
  state.role = role;
  localStorage.setItem("token", output.token);
  localStorage.setItem("role", role);
}

async function withLoading(task) {
  if (state.loading) return;
  state.loading = true;
  render();
  try {
    await task();
  } catch (err) {
    showToast(err.message, "error");
  } finally {
    state.loading = false;
    render();
  }
}

async function bootstrapRoleData() {
  if (!state.token) return;
  if (state.role === "BUYER") {
    await refreshBuyerProfile();
    await refreshBuyerOrders();
  } else {
    await refreshSellerProfile();
    await Promise.all([refreshSellerProducts(), refreshTrend()]);
  }
}

async function refreshBuyerProfile() {
  state.buyerProfile = await apiFetch("/api/buyer/profile");
}

async function refreshSellerProfile() {
  state.sellerProfile = await apiFetch("/api/seller/profile");
  if (state.sellerProfile.theme) {
    state.theme = state.sellerProfile.theme;
    localStorage.setItem("theme", state.theme);
    document.body.classList.toggle("dark", state.theme === "DARK");
  }
}

async function refreshBuyerProducts(params = {}) {
  const namePrefix = params.namePrefix || params.buyerProductName || state.buyerProductName;
  state.buyerProductName = namePrefix;
  if (!namePrefix.trim()) {
    showToast("请输入商品名前缀");
    return;
  }
  const data = await apiFetch(`/api/buyer/products${qs({ namePrefix, page: 1, pageSize: 20 })}`);
  state.buyerProducts = data.items || [];
}

async function refreshBuyerOrders(params = {}) {
  state.buyerOrderStatus = params.status ?? state.buyerOrderStatus;
  const data = await apiFetch(`/api/buyer/orders${qs({ status: state.buyerOrderStatus, page: 1, pageSize: 50 })}`);
  state.buyerOrders = data.items || [];
}

async function refreshSellerProducts(params = {}) {
  const filter = validateSellerProductFilter(params);
  const data = await apiFetch(`/api/seller/products${qs({ ...filter, page: 1, pageSize: 50 })}`);
  state.sellerProducts = data.items || [];
}

function validateSellerProductFilter(params = {}) {
  const startDate = params.startDate || "";
  const endDate = params.endDate || "";
  if (!startDate && !endDate) return params;
  const today = localDateString();
  if (!startDate || !endDate) {
    throw new Error("商品筛选需要同时选择开始日期和结束日期");
  }
  if (endDate > today || startDate > today) {
    throw new Error("商品筛选只能查看今天及以前的数据");
  }
  if (startDate > endDate) {
    throw new Error("开始日期不能晚于结束日期");
  }
  return params;
}

async function refreshSellerOrders(params = {}) {
  const productId = params.productId ?? state.sellerOrderProductId;
  const productNamePrefix = params.productNamePrefix ?? state.sellerOrderProductNamePrefix;
  const filter = validateSellerOrderFilter({
    productId,
    productNamePrefix,
  });
  state.sellerOrderProductId = filter.productId;
  state.sellerOrderProductNamePrefix = filter.productNamePrefix;
  const data = await apiFetch(`/api/seller/orders${qs({ ...filter, page: 1, pageSize: 50 })}`);
  state.sellerOrders = data.items || [];
}

function validateSellerOrderFilter(params = {}) {
  const productId = String(params.productId || "").trim();
  const productNamePrefix = String(params.productNamePrefix || "").trim();
  if (!productId && !productNamePrefix) {
    throw new Error("请输入商品 ID 或商品名前缀");
  }
  if (productId && (!Number.isInteger(Number(productId)) || Number(productId) <= 0)) {
    throw new Error("商品 ID 不合法");
  }
  return { productId, productNamePrefix };
}

async function refreshTrend(input = {}) {
  const options = typeof input === "number" ? { days: input } : input;
  const range = normalizeTrendRange(options);
  const data = await apiFetch(`/api/seller/trends${qs({ startDate: range.startDate, endDate: range.endDate })}`);
  state.trendStartDate = range.startDate;
  state.trendEndDate = range.endDate;
  state.trendPoints = data.points || [];
  state.trendHoverIndex = -1;
  hideTrendTooltip();
  window.setTimeout(drawTrendChart, 0);
}

function normalizeTrendRange(options = {}) {
  const today = localDateString();
  const days = Number(options.days) > 0 ? Math.min(Number(options.days), 90) : 7;
  let startDate = options.startDate ?? "";
  let endDate = options.endDate ?? "";
  if (!startDate && !endDate) {
    endDate = state.trendEndDate || today;
    startDate = state.trendStartDate || addDays(endDate, -days + 1);
  }
  if (!startDate || !endDate) {
    throw new Error("请选择完整的日期范围");
  }
  if (endDate > today) {
    throw new Error("只能查看今天及以前的数据");
  }
  if (startDate > endDate) {
    throw new Error("开始日期不能晚于结束日期");
  }
  if (daysBetween(startDate, endDate) > 90) {
    throw new Error("日期范围不能超过 90 天");
  }
  return { startDate, endDate };
}

function localDateString(date = new Date()) {
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return local.toISOString().slice(0, 10);
}

function dateToUTC(dateText) {
  const [year, month, day] = String(dateText).split("-").map(Number);
  return Date.UTC(year, month - 1, day);
}

function addDays(dateText, offset) {
  const date = new Date(dateToUTC(dateText) + offset * 86400000);
  return date.toISOString().slice(0, 10);
}

function daysBetween(startDate, endDate) {
  return Math.floor((dateToUTC(endDate) - dateToUTC(startDate)) / 86400000) + 1;
}

function render() {
  if (!state.token) {
    renderAuth();
    return;
  }
  renderWorkspace();
}

function renderAuth() {
  app.innerHTML = `
    <section class="auth-layout">
      <div class="hero-panel">
        <span class="eyebrow">Microservice Commerce Console</span>
        <h1 class="hero-title">一套前端打通买家与商家交易闭环</h1>
        <p class="hero-copy">从注册登录、商品浏览、下单支付，到发货、退款、确认收货和自动下架，用最少页面覆盖完整电商 Demo。</p>
        <div class="metric-strip">
          <div class="metric"><strong>2</strong><span>用户角色</span></div>
          <div class="metric"><strong>5</strong><span>核心页面</span></div>
          <div class="metric"><strong>20+</strong><span>后端接口</span></div>
        </div>
      </div>
      <div class="card auth-card">
        <div class="section-title">
          <div>
            <h2>${state.authMode === "login" ? "登录" : "创建账号"}</h2>
            <span class="subtle">选择身份后进入对应工作台</span>
          </div>
        </div>
        <div class="role-switch" data-group="role">
          <button data-action="set-role" data-role="BUYER" class="${state.role === "BUYER" ? "active" : ""}">买家</button>
          <button data-action="set-role" data-role="SELLER" class="${state.role === "SELLER" ? "active" : ""}">商家</button>
        </div>
        <div class="segmented mt">
          <button data-action="set-auth-mode" data-mode="login" class="${state.authMode === "login" ? "active" : ""}">登录</button>
          <button data-action="set-auth-mode" data-mode="register" class="${state.authMode === "register" ? "active" : ""}">注册</button>
        </div>
        <form class="form-grid mt" data-form="${state.authMode}">
          <div class="field full">
            <label>API 地址，留空表示同源</label>
            <input name="apiBase" value="${htmlEscape(state.apiBase)}" placeholder="例如 http://127.0.0.1:8080" />
          </div>
          <div class="field">
            <label>用户名</label>
            <input name="username" required placeholder="buyer001" />
          </div>
          <div class="field">
            <label>密码</label>
            <input name="password" type="password" required placeholder="请输入密码" />
          </div>
          ${state.authMode === "register" ? renderRegisterFields() : ""}
          <button class="primary field full" type="submit">${state.authMode === "login" ? "登录进入" : "创建并继续登录"}</button>
        </form>
      </div>
    </section>
  `;
}

function renderRegisterFields() {
  if (state.role === "BUYER") {
    return `
      <div class="field"><label>昵称</label><input name="nickname" placeholder="小明" /></div>
      <div class="field"><label>手机号</label><input name="phone" placeholder="13800000000" /></div>
      <div class="field full"><label>收货地址</label><textarea name="shippingAddress" placeholder="北京市朝阳区..."></textarea></div>
      <div class="field full"><label>头像 URL</label><input name="avatarUrl" placeholder="可选" /></div>
    `;
  }
  return `
    <div class="field"><label>注册人</label><input name="registrantName" required placeholder="张三" /></div>
    <div class="field"><label>店铺名</label><input name="shopName" required placeholder="张三小店" /></div>
    <div class="field full"><label>店铺头像 URL</label><input name="shopAvatarUrl" placeholder="可选" /></div>
  `;
}

function renderWorkspace() {
  const isBuyer = state.role === "BUYER";
  app.innerHTML = `
    <section class="workspace">
      <header class="topbar">
        <div class="brand">
          <span class="eyebrow">${isBuyer ? "Buyer Console" : "Seller Console"}</span>
          <h1>${isBuyer ? "买家工作台" : "商家工作台"}</h1>
        </div>
        <div class="top-actions">
          <button class="ghost" data-action="toggle-theme">${state.theme === "DARK" ? "浅色主题" : "深色主题"}</button>
          <button class="danger" data-action="logout">退出登录</button>
        </div>
      </header>
      <div class="main-grid">
        <aside class="sidebar">
          ${isBuyer ? renderBuyerTabs() : renderSellerTabs()}
        </aside>
        <section class="content">
          ${state.loading ? `<div class="card subtle">处理中...</div>` : ""}
          ${isBuyer ? renderBuyerContent() : renderSellerContent()}
        </section>
      </div>
    </section>
  `;
  if (!isBuyer && state.sellerTab === "products") {
    window.setTimeout(drawTrendChart, 0);
  }
}

function renderBuyerTabs() {
  return `
    <nav class="tabs">
      <button data-action="buyer-tab" data-tab="products" class="${state.buyerTab === "products" ? "active" : ""}">商品浏览</button>
      <button data-action="buyer-tab" data-tab="orders" class="${state.buyerTab === "orders" ? "active" : ""}">订单管理</button>
      <button data-action="buyer-tab" data-tab="settings" class="${state.buyerTab === "settings" ? "active" : ""}">设置</button>
    </nav>
  `;
}

function renderSellerTabs() {
  return `
    <nav class="tabs">
      <button data-action="seller-tab" data-tab="products" class="${state.sellerTab === "products" ? "active" : ""}">商品管理</button>
      <button data-action="seller-tab" data-tab="orders" class="${state.sellerTab === "orders" ? "active" : ""}">订单查询</button>
      <button data-action="seller-tab" data-tab="settings" class="${state.sellerTab === "settings" ? "active" : ""}">设置</button>
    </nav>
  `;
}

function renderBuyerContent() {
  if (state.buyerTab === "orders") return renderBuyerOrders();
  if (state.buyerTab === "settings") return renderBuyerSettings();
  return renderBuyerProducts();
}

function renderSellerContent() {
  if (state.sellerTab === "orders") return renderSellerOrders();
  if (state.sellerTab === "settings") return renderSellerSettings();
  return renderSellerProducts();
}

function renderBuyerProducts() {
  return `
    <div class="card">
      <div class="section-title">
        <div><h2>商品浏览</h2><span class="subtle">按商品名前缀搜索后下单支付</span></div>
      </div>
      <form class="toolbar" data-form="buyer-search-products">
        <div class="field"><label>商品名前缀</label><input name="buyerProductName" value="${htmlEscape(state.buyerProductName)}" placeholder="phone" /></div>
        <button class="primary" type="submit">搜索商品</button>
      </form>
    </div>
    <div class="card">
      <div class="table-wrap">
        <table>
          <thead><tr><th>商品</th><th>店铺</th><th>价格</th><th>状态</th><th>展示库存</th><th>操作</th></tr></thead>
          <tbody>${state.buyerProducts.map(renderBuyerProductRow).join("") || emptyRow(6, "暂无商品，请先搜索")}</tbody>
        </table>
      </div>
    </div>
  `;
}

function renderBuyerProductRow(item) {
  return `
    <tr>
      <td><strong>${htmlEscape(item.productName)}</strong><br><span class="subtle">${htmlEscape(item.description)}</span><br><span class="copyable" data-copy="${item.productId}">ID ${item.productId}</span></td>
      <td>${htmlEscape(item.shopName || "-")}</td>
      <td>${money(item.priceCent)}</td>
      <td>${statusBadge(item.status, PRODUCT_STATUS[item.status])}</td>
      <td>${item.displayInventory}</td>
      <td>
        <form class="row-actions" data-form="create-order" data-product-id="${item.productId}">
          <input name="quantity" type="number" min="1" max="100" value="1" />
          <button class="primary" type="submit">下单支付</button>
        </form>
      </td>
    </tr>
  `;
}

function renderBuyerOrders() {
  return `
    <div class="card">
      <div class="section-title">
        <div><h2>订单管理</h2><span class="subtle">默认展示已下单未发货和配送中订单</span></div>
      </div>
      <form class="toolbar" data-form="buyer-search-orders">
        <div class="field">
          <label>订单状态</label>
          <select name="status">
            <option value="" ${state.buyerOrderStatus === "" ? "selected" : ""}>默认</option>
            <option value="1" ${state.buyerOrderStatus === "1" ? "selected" : ""}>已下单未发货</option>
            <option value="2" ${state.buyerOrderStatus === "2" ? "selected" : ""}>配送中</option>
            <option value="3" ${state.buyerOrderStatus === "3" ? "selected" : ""}>已收货</option>
            <option value="4" ${state.buyerOrderStatus === "4" ? "selected" : ""}>已退款</option>
          </select>
        </div>
        <button class="primary" type="submit">查询订单</button>
      </form>
    </div>
    <div class="card">
      <div class="table-wrap">
        <table>
          <thead><tr><th>订单</th><th>商品</th><th>数量</th><th>金额</th><th>退款金额</th><th>状态</th><th>创建时间</th><th>操作</th></tr></thead>
          <tbody>${state.buyerOrders.map(renderBuyerOrderRow).join("") || emptyRow(8, "暂无订单")}</tbody>
        </table>
      </div>
    </div>
  `;
}

function renderBuyerOrderRow(item) {
  return `
    <tr>
      <td><span class="copyable" data-copy="${item.orderId}">#${item.orderId}</span></td>
      <td>${htmlEscape(item.productNameSnapshot)}<br><span class="subtle">商品 ID ${item.productId}</span></td>
      <td>${item.quantity}</td>
      <td>${money(item.totalAmountCent)}</td>
      <td>${money(item.refundAmountCent)}</td>
      <td>${statusBadge(item.status, ORDER_STATUS[item.status])}</td>
      <td>${htmlEscape(item.createdAt)}</td>
      <td>
        <div class="row-actions">
          ${item.status === 1 ? `<button class="danger" data-action="refund-order" data-order-id="${item.orderId}">退款</button>` : ""}
          ${item.status === 2 ? `<button class="secondary" data-action="receive-order" data-order-id="${item.orderId}">确认收货</button>` : ""}
          ${item.status !== 1 && item.status !== 2 ? `<span class="subtle">无可用操作</span>` : ""}
        </div>
      </td>
    </tr>
  `;
}

function renderBuyerSettings() {
  const p = state.buyerProfile || {};
  return `
    <div class="dashboard-cards">
      <div class="stat-card"><span class="subtle">用户 ID</span><strong>${p.userId || "-"}</strong></div>
      <div class="stat-card"><span class="subtle">当前余额</span><strong>${money(p.balanceCent)}</strong></div>
      <div class="stat-card"><span class="subtle">角色</span><strong>买家</strong></div>
    </div>
    <div class="card">
      <div class="section-title"><h2>买家资料</h2></div>
      <form class="form-grid" data-form="buyer-profile">
        <div class="field"><label>昵称</label><input name="nickname" value="${htmlEscape(p.nickname)}" required /></div>
        <div class="field"><label>手机号</label><input name="phone" value="${htmlEscape(p.phone)}" /></div>
        <div class="field full"><label>头像 URL</label><input name="avatarUrl" value="${htmlEscape(p.avatarUrl)}" /></div>
        <div class="field full"><label>收货地址</label><textarea name="shippingAddress">${htmlEscape(p.shippingAddress)}</textarea></div>
        <button class="primary field full" type="submit">保存资料</button>
      </form>
    </div>
    <div class="card">
      <div class="section-title"><h2>余额充值</h2></div>
      <form class="toolbar" data-form="buyer-recharge">
        <div class="field"><label>充值金额，单位元</label><input name="amount" type="number" min="0.01" step="0.01" value="100" /></div>
        <button class="secondary" type="submit">充值</button>
      </form>
    </div>
  `;
}

function renderSellerProducts() {
  const today = localDateString();
  const trendEndDate = state.trendEndDate || today;
  const trendStartDate = state.trendStartDate || addDays(trendEndDate, -6);
  return `
    <div class="dashboard-cards">
      <div class="stat-card"><span class="subtle">店铺成交总额</span><strong>${money(state.sellerProfile?.totalDealAmountCent)}</strong></div>
      <div class="stat-card"><span class="subtle">当前商品数</span><strong>${state.sellerProducts.length}</strong></div>
      <div class="stat-card"><span class="subtle">店铺</span><strong>${htmlEscape(state.sellerProfile?.shopName || "-")}</strong></div>
    </div>
    <div class="card">
      <div class="section-title">
        <div><h2>趋势图</h2><span class="subtle">成交金额、退款金额、退款率</span></div>
        <form class="trend-range" data-form="seller-trend">
          <div class="field">
            <label>开始日期</label>
            <input name="startDate" type="date" max="${today}" value="${trendStartDate}" required />
          </div>
          <div class="field">
            <label>结束日期</label>
            <input name="endDate" type="date" max="${today}" value="${trendEndDate}" required />
          </div>
          <button class="ghost" type="submit">刷新趋势</button>
        </form>
      </div>
      <div class="chart-panel">
        <div class="chart-legend">
          <span><b style="background:#b96b24"></b>成交额</span>
          <span><b style="background:#b73535"></b>退款额</span>
          <span><b style="background:#1f5b4e"></b>退款率</span>
        </div>
        <canvas id="trendChart" class="chart"></canvas>
        <div id="trendTooltip" class="chart-tooltip hidden"></div>
      </div>
    </div>
    <div class="card">
      <div class="section-title"><h2>创建商品</h2></div>
      <form class="toolbar" data-form="seller-create-product">
        <div class="field"><label>商品名</label><input name="productName" required /></div>
        <div class="field"><label>价格，单位元</label><input name="price" type="number" min="0.01" step="0.01" required /></div>
        <div class="field"><label>初始库存</label><input name="initialInventory" type="number" min="0" value="0" /></div>
        <div class="field"><label>状态</label>${productCreateStatusSelect("status", 1)}</div>
        <div class="field"><label>描述</label><input name="description" /></div>
        <button class="primary" type="submit">创建商品</button>
      </form>
    </div>
    <div class="card">
      <div class="section-title"><h2>商品筛选</h2></div>
      <form class="toolbar" data-form="seller-search-products">
        <div class="field"><label>开始日期</label><input name="startDate" type="date" max="${today}" /></div>
        <div class="field"><label>结束日期</label><input name="endDate" type="date" max="${today}" /></div>
        <div class="field"><label>状态</label>${productStatusSelect("status", "")}</div>
        <div class="field"><label>商品 ID</label><input name="productId" type="number" /></div>
        <div class="field"><label>商品名前缀</label><input name="productNamePrefix" /></div>
        <div class="field"><label>发货状态</label><select name="shipped"><option value="">全部</option><option value="false">待发货</option><option value="true">已发货/完成</option></select></div>
        <button class="primary" type="submit">查询商品</button>
      </form>
    </div>
    <div class="card">
      <div class="table-wrap">
        <table>
          <thead><tr><th>商品</th><th>价格</th><th>状态</th><th>库存</th><th>成交额</th><th>退款额</th><th>退款率</th><th>操作</th></tr></thead>
          <tbody>${state.sellerProducts.map(renderSellerProductRow).join("") || emptyRow(8, "暂无商品")}</tbody>
        </table>
      </div>
    </div>
  `;
}

function renderSellerProductRow(item) {
  return `
    <tr>
      <td><strong>${htmlEscape(item.productName)}</strong><br><span class="subtle">${htmlEscape(item.description)}</span><br><span class="copyable" data-copy="${item.productId}">ID ${item.productId}</span></td>
      <td>${money(item.priceCent)}</td>
      <td>${statusBadge(item.status, PRODUCT_STATUS[item.status])}</td>
      <td>${item.displayInventory}</td>
      <td>${money(item.dealAmountCent)}</td>
      <td>${money(item.refundAmountCent)}</td>
      <td>${((Number(item.refundRate) || 0) * 100).toFixed(2)}%</td>
      <td>
        <div class="row-actions">
          <button class="ghost" data-action="edit-product" data-product-id="${item.productId}">编辑</button>
          <button class="ghost" data-action="view-product-orders" data-product-id="${item.productId}">查订单</button>
          <button class="secondary" data-action="add-inventory" data-product-id="${item.productId}">补库存</button>
          <button class="ghost" data-action="ship-product" data-product-id="${item.productId}">一键发货</button>
          ${item.status === 1 ? `<button class="danger" data-action="delist-product" data-product-id="${item.productId}">下架</button>` : ""}
        </div>
      </td>
    </tr>
  `;
}

function renderSellerOrders() {
  return `
    <div class="card">
      <div class="section-title">
        <div><h2>订单查询</h2><span class="subtle">输入商品 ID 或商品名前缀，查看该商品订单</span></div>
      </div>
      <form class="toolbar" data-form="seller-search-orders">
        <div class="field"><label>商品 ID</label><input name="productId" type="number" min="1" value="${htmlEscape(state.sellerOrderProductId)}" placeholder="3001" /></div>
        <div class="field"><label>商品名前缀</label><input name="productNamePrefix" value="${htmlEscape(state.sellerOrderProductNamePrefix)}" placeholder="phone" /></div>
        <button class="primary" type="submit">查询订单</button>
      </form>
    </div>
    <div class="card">
      <div class="table-wrap">
        <table class="orders-table">
          <thead><tr><th>订单</th><th>商品</th><th>买家</th><th>数量</th><th>金额</th><th>退款金额</th><th>状态</th><th>时间</th><th>操作</th></tr></thead>
          <tbody>${state.sellerOrders.map(renderSellerOrderRow).join("") || emptyRow(9, "暂无订单，请输入商品 ID 或商品名前缀查询")}</tbody>
        </table>
      </div>
    </div>
  `;
}

function renderSellerOrderRow(item) {
  const timeline = [
    ["创建", item.createdAt],
    ["发货", item.shippedAt],
    ["收货", item.receivedAt],
    ["退款", item.refundedAt],
  ].filter(([, value]) => value).map(([label, value]) => `<span>${label} ${htmlEscape(value)}</span>`).join("<br>");
  return `
    <tr>
      <td><span class="copyable" data-copy="${item.orderId}">#${item.orderId}</span></td>
      <td>${htmlEscape(item.productNameSnapshot)}<br><span class="copyable" data-copy="${item.productId}">商品 ID ${item.productId}</span></td>
      <td><span class="copyable" data-copy="${item.buyerId}">买家 ID ${item.buyerId}</span></td>
      <td>${item.quantity}</td>
      <td>${money(item.totalAmountCent)}</td>
      <td>${money(item.refundAmountCent)}</td>
      <td>${statusBadge(item.status, ORDER_STATUS[item.status])}</td>
      <td>${timeline || "-"}</td>
      <td>
        <div class="row-actions">
          ${item.status === 1 ? `<button class="secondary" data-action="ship-order" data-order-id="${item.orderId}">发货</button>` : `<span class="subtle">无可用操作</span>`}
        </div>
      </td>
    </tr>
  `;
}

function renderSellerSettings() {
  const p = state.sellerProfile || {};
  return `
    <div class="dashboard-cards">
      <div class="stat-card"><span class="subtle">商家 ID</span><strong>${p.userId || "-"}</strong></div>
      <div class="stat-card"><span class="subtle">成交商品总额</span><strong>${money(p.totalDealAmountCent)}</strong></div>
      <div class="stat-card"><span class="subtle">主题</span><strong>${state.theme}</strong></div>
    </div>
    <div class="card">
      <div class="section-title"><h2>商家资料</h2></div>
      <form class="form-grid" data-form="seller-profile">
        <div class="field"><label>注册人</label><input name="registrantName" value="${htmlEscape(p.registrantName)}" required /></div>
        <div class="field"><label>店铺名</label><input name="shopName" value="${htmlEscape(p.shopName)}" required /></div>
        <div class="field"><label>主题</label><select name="theme"><option value="LIGHT" ${state.theme === "LIGHT" ? "selected" : ""}>LIGHT</option><option value="DARK" ${state.theme === "DARK" ? "selected" : ""}>DARK</option></select></div>
        <div class="field full"><label>店铺头像 URL</label><input name="shopAvatarUrl" value="${htmlEscape(p.shopAvatarUrl)}" /></div>
        <button class="primary field full" type="submit">保存店铺资料</button>
      </form>
    </div>
  `;
}

function productStatusSelect(name, selected) {
  return `
    <select name="${name}">
      <option value="" ${selected === "" ? "selected" : ""}>全部</option>
      <option value="1" ${Number(selected) === 1 ? "selected" : ""}>在售</option>
      <option value="2" ${Number(selected) === 2 ? "selected" : ""}>预售中</option>
      <option value="3" ${Number(selected) === 3 ? "selected" : ""}>正在下架</option>
      <option value="4" ${Number(selected) === 4 ? "selected" : ""}>已下架</option>
    </select>
  `;
}

function productCreateStatusSelect(name, selected) {
  return `
    <select name="${name}">
      <option value="1" ${Number(selected) === 1 ? "selected" : ""}>在售</option>
      <option value="2" ${Number(selected) === 2 ? "selected" : ""}>预售中</option>
    </select>
  `;
}

function statusBadge(status, label) {
  const cls = status === 1 || status === 2 ? "ok" : status === 3 ? "warn" : "danger";
  return `<span class="badge ${cls}">${label || status}</span>`;
}

function emptyRow(colspan, text) {
  return `<tr><td colspan="${colspan}" class="empty">${text}</td></tr>`;
}

function drawTrendChart() {
  const canvas = document.querySelector("#trendChart");
  if (!canvas) return;
  const rect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  canvas.width = rect.width * dpr;
  canvas.height = rect.height * dpr;
  const ctx = canvas.getContext("2d");
  ctx.scale(dpr, dpr);
  ctx.clearRect(0, 0, rect.width, rect.height);
  const points = state.trendPoints;
  const colors = cssColors();
  const layout = trendLayout(rect);
  const maxAmountCent = niceMaxCent(Math.max(...points.flatMap((p) => [p.dealAmountCent || 0, p.refundAmountCent || 0]), 0));
  drawTrendAxes(ctx, rect, layout, maxAmountCent, colors);
  if (!points.length) {
    ctx.fillStyle = colors.muted;
    ctx.fillText("暂无趋势数据", layout.left, layout.top + 22);
    return;
  }
  drawTrendLine(ctx, layout, points, "dealAmountCent", maxAmountCent, "#b96b24", false);
  drawTrendLine(ctx, layout, points, "refundAmountCent", maxAmountCent, "#b73535", false);
  drawTrendLine(ctx, layout, points, "refundRate", 1, "#1f5b4e", true);
  drawTrendHover(ctx, layout, points, maxAmountCent, colors);
  setupTrendChartInteractions(canvas, layout, points);
}

function trendLayout(rect) {
  return {
    left: 72,
    right: rect.width - 72,
    top: 34,
    bottom: rect.height - 46,
  };
}

function cssColors() {
  const styles = getComputedStyle(document.body);
  return {
    text: styles.getPropertyValue("--text").trim(),
    muted: styles.getPropertyValue("--muted").trim(),
    line: styles.getPropertyValue("--line").trim(),
  };
}

function niceMaxCent(value) {
  const amount = Math.max(Number(value) || 0, 100);
  const magnitude = 10 ** Math.floor(Math.log10(amount));
  const normalized = amount / magnitude;
  const niceNormalized = normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10;
  return niceNormalized * magnitude;
}

function drawTrendAxes(ctx, rect, layout, maxAmountCent, colors) {
  ctx.font = "12px Georgia, serif";
  ctx.textBaseline = "middle";
  ctx.lineWidth = 1;
  for (let i = 0; i <= 4; i += 1) {
    const ratio = i / 4;
    const value = maxAmountCent * (1 - ratio);
    const y = layout.top + ratio * (layout.bottom - layout.top);
    ctx.strokeStyle = ratio === 1 ? colors.text : colors.line;
    ctx.beginPath();
    ctx.moveTo(layout.left, y);
    ctx.lineTo(layout.right, y);
    ctx.stroke();
    ctx.fillStyle = colors.muted;
    ctx.textAlign = "right";
    ctx.fillText(money(value), layout.left - 10, y);
    ctx.textAlign = "left";
    ctx.fillText(`${Math.round((1 - ratio) * 100)}%`, layout.right + 10, y);
  }
  ctx.strokeStyle = colors.text;
  ctx.beginPath();
  ctx.moveTo(layout.left, layout.top);
  ctx.lineTo(layout.left, layout.bottom);
  ctx.lineTo(layout.right, layout.bottom);
  ctx.stroke();

  ctx.fillStyle = colors.muted;
  ctx.textAlign = "center";
  ctx.textBaseline = "top";
  state.trendPoints.forEach((point, index) => {
    const x = trendX(layout, state.trendPoints.length, index);
    ctx.fillText(formatTrendDate(point.date), x, layout.bottom + 12);
  });
}

function drawTrendLine(ctx, layout, points, key, maxValue, color, dashed) {
  ctx.strokeStyle = color;
  ctx.lineWidth = 2;
  if (dashed) ctx.setLineDash([8, 6]);
  else ctx.setLineDash([]);
  ctx.beginPath();
  points.forEach((point, index) => {
    const x = trendX(layout, points.length, index);
    const y = trendY(layout, Number(point[key]) || 0, maxValue);
    if (index === 0) ctx.moveTo(x, y);
    else ctx.lineTo(x, y);
  });
  ctx.stroke();
  ctx.setLineDash([]);
}

function drawTrendHover(ctx, layout, points, maxAmountCent, colors) {
  const index = state.trendHoverIndex;
  if (index < 0 || !points[index]) return;
  const point = points[index];
  const x = trendX(layout, points.length, index);
  ctx.strokeStyle = colors.text;
  ctx.setLineDash([5, 5]);
  ctx.beginPath();
  ctx.moveTo(x, layout.top);
  ctx.lineTo(x, layout.bottom);
  ctx.stroke();
  ctx.setLineDash([]);
  [
    ["dealAmountCent", maxAmountCent, "#b96b24"],
    ["refundAmountCent", maxAmountCent, "#b73535"],
    ["refundRate", 1, "#1f5b4e"],
  ].forEach(([key, maxValue, color]) => {
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.arc(x, trendY(layout, Number(point[key]) || 0, maxValue), 4, 0, Math.PI * 2);
    ctx.fill();
  });
}

function setupTrendChartInteractions(canvas, layout, points) {
  const tooltip = document.querySelector("#trendTooltip");
  if (!tooltip) return;
  canvas.onmousemove = (event) => {
    if (!points.length) return;
    const rect = canvas.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const step = points.length === 1 ? 0 : (layout.right - layout.left) / (points.length - 1);
    const index = points.length === 1 ? 0 : Math.round((x - layout.left) / step);
    if (index < 0 || index >= points.length) {
      hideTrendTooltip();
      return;
    }
    state.trendHoverIndex = index;
    const point = points[index];
    const pointX = trendX(layout, points.length, index);
    tooltip.innerHTML = `
      <strong>${htmlEscape(point.date)}</strong><br>
      成交额：${money(point.dealAmountCent)}<br>
      退款额：${money(point.refundAmountCent)}<br>
      退款率：${((Number(point.refundRate) || 0) * 100).toFixed(2)}%
    `;
    tooltip.style.left = `${Math.min(Math.max(pointX + 12, 12), rect.width - 180)}px`;
    tooltip.style.top = "46px";
    tooltip.classList.remove("hidden");
    drawTrendChart();
  };
  canvas.onmouseleave = () => {
    state.trendHoverIndex = -1;
    hideTrendTooltip();
    drawTrendChart();
  };
}

function hideTrendTooltip() {
  const tooltip = document.querySelector("#trendTooltip");
  if (tooltip) tooltip.classList.add("hidden");
}

function trendX(layout, count, index) {
  return count === 1 ? layout.left : layout.left + (index / (count - 1)) * (layout.right - layout.left);
}

function trendY(layout, value, maxValue) {
  return layout.bottom - (Math.max(value, 0) / Math.max(maxValue, 1)) * (layout.bottom - layout.top);
}

function formatTrendDate(date) {
  return String(date || "").slice(5) || "-";
}

document.addEventListener("submit", (event) => {
  const form = event.target.closest("form[data-form]");
  if (!form) return;
  event.preventDefault();
  const data = formData(form);
  const type = form.dataset.form;
  withLoading(async () => {
    if (type === "login") await submitLogin(data);
    if (type === "register") await submitRegister(data);
    if (type === "buyer-search-products") await refreshBuyerProducts(data);
    if (type === "create-order") await submitCreateOrder(form, data);
    if (type === "buyer-search-orders") await refreshBuyerOrders(data);
    if (type === "buyer-profile") await submitBuyerProfile(data);
    if (type === "buyer-recharge") await submitRecharge(data);
    if (type === "seller-create-product") await submitCreateProduct(data);
    if (type === "seller-search-products") await refreshSellerProducts(data);
    if (type === "seller-search-orders") await refreshSellerOrders(data);
    if (type === "seller-trend") await refreshTrend({ startDate: data.startDate, endDate: data.endDate });
    if (type === "seller-profile") await submitSellerProfile(data);
  });
});

document.addEventListener("click", (event) => {
  const target = event.target.closest("[data-action], [data-copy]");
  if (!target) return;
  if (target.dataset.copy) {
    navigator.clipboard?.writeText(target.dataset.copy);
    showToast("已复制");
    return;
  }
  const action = target.dataset.action;
  withLoading(async () => {
    if (action === "set-role") {
      state.role = target.dataset.role;
      localStorage.setItem("role", state.role);
    }
    if (action === "set-auth-mode") state.authMode = target.dataset.mode;
    if (action === "buyer-tab") {
      state.buyerTab = target.dataset.tab;
      if (state.buyerTab === "orders") await refreshBuyerOrders();
      if (state.buyerTab === "settings") await refreshBuyerProfile();
    }
    if (action === "seller-tab") {
      state.sellerTab = target.dataset.tab;
      if (state.sellerTab === "products") await Promise.all([refreshSellerProducts(), refreshTrend()]);
      if (state.sellerTab === "settings") await refreshSellerProfile();
    }
    if (action === "toggle-theme") await toggleTheme();
    if (action === "logout") await logout();
    if (action === "refund-order") await mutateOrder(`/api/buyer/orders/${target.dataset.orderId}/refund`, "refund");
    if (action === "receive-order") await mutateOrder(`/api/buyer/orders/${target.dataset.orderId}/receive`);
    if (action === "ship-order") await shipSellerOrder(target.dataset.orderId);
    if (action === "add-inventory") await addInventory(target.dataset.productId);
    if (action === "ship-product") await shipProduct(target.dataset.productId);
    if (action === "delist-product") await delistProduct(target.dataset.productId);
    if (action === "edit-product") await editProduct(target.dataset.productId);
    if (action === "view-product-orders") {
      state.sellerTab = "orders";
      await refreshSellerOrders({ productId: target.dataset.productId, productNamePrefix: "" });
    }
  });
});

async function submitLogin(data) {
  state.apiBase = data.apiBase.trim();
  localStorage.setItem("apiBase", state.apiBase);
  const output = await apiFetch("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ username: data.username, password: data.password, role: state.role }),
  });
  setSession(output, state.role);
  await bootstrapRoleData();
  showToast("登录成功");
}

async function submitRegister(data) {
  state.apiBase = data.apiBase.trim();
  localStorage.setItem("apiBase", state.apiBase);
  const profile = state.role === "BUYER"
    ? {
        nickname: data.nickname,
        avatarUrl: data.avatarUrl,
        phone: data.phone,
        shippingAddress: data.shippingAddress,
      }
    : {
        registrantName: data.registrantName,
        shopName: data.shopName,
        shopAvatarUrl: data.shopAvatarUrl,
      };
  await apiFetch("/api/auth/register", {
    method: "POST",
    body: JSON.stringify({ username: data.username, password: data.password, role: state.role, profile }),
  });
  showToast("注册成功，请继续登录");
  state.authMode = "login";
}

async function submitCreateOrder(form, data) {
  const productId = Number(form.dataset.productId);
  const quantity = Number(data.quantity);
  await apiFetch("/api/buyer/orders", {
    method: "POST",
    idempotent: "order",
    body: JSON.stringify({ productId, quantity }),
  });
  await Promise.all([refreshBuyerProfile(), refreshBuyerOrders()]);
  showToast("下单成功");
}

async function submitBuyerProfile(data) {
  await apiFetch("/api/buyer/profile", {
    method: "PUT",
    body: JSON.stringify(data),
  });
  await refreshBuyerProfile();
  showToast("买家资料已保存");
}

async function submitRecharge(data) {
  const amountCent = Math.round(Number(data.amount) * 100);
  await apiFetch("/api/buyer/balance/recharge", {
    method: "POST",
    idempotent: "recharge",
    body: JSON.stringify({ amountCent }),
  });
  await refreshBuyerProfile();
  showToast("充值成功");
}

async function submitCreateProduct(data) {
  await apiFetch("/api/seller/products", {
    method: "POST",
    body: JSON.stringify({
      productName: data.productName,
      description: data.description,
      priceCent: Math.round(Number(data.price) * 100),
      status: Number(data.status),
      initialInventory: Number(data.initialInventory) || 0,
    }),
  });
  await refreshSellerProducts();
  showToast("商品已创建");
}

async function submitSellerProfile(data) {
  await apiFetch("/api/seller/profile", {
    method: "PUT",
    body: JSON.stringify(data),
  });
  await refreshSellerProfile();
  showToast("店铺资料已保存");
}

async function toggleTheme() {
  state.theme = state.theme === "DARK" ? "LIGHT" : "DARK";
  localStorage.setItem("theme", state.theme);
  document.body.classList.toggle("dark", state.theme === "DARK");
  if (state.role === "SELLER" && state.sellerProfile) {
    await submitSellerProfile({
      registrantName: state.sellerProfile.registrantName,
      shopName: state.sellerProfile.shopName,
      shopAvatarUrl: state.sellerProfile.shopAvatarUrl,
      theme: state.theme,
    });
  }
}

async function logout() {
  try {
    await apiFetch("/api/auth/logout", { method: "POST" });
  } finally {
    localStorage.removeItem("token");
    state.token = "";
    state.buyerProfile = null;
    state.sellerProfile = null;
    showToast("已退出登录");
  }
}

async function mutateOrder(path, prefix) {
  await apiFetch(path, {
    method: "POST",
    ...(prefix ? { idempotent: prefix } : {}),
  });
  await refreshBuyerOrders();
  showToast("订单已更新");
}

async function addInventory(productId) {
  const quantity = Number(window.prompt("请输入补充库存数量", "10"));
  if (!quantity || quantity <= 0) return;
  await apiFetch(`/api/seller/products/${productId}/inventory/add`, {
    method: "POST",
    body: JSON.stringify({ quantity }),
  });
  await refreshSellerProducts();
  showToast("库存已补充");
}

async function shipProduct(productId) {
  await apiFetch(`/api/seller/products/${productId}/ship-all`, { method: "POST" });
  await refreshSellerProducts();
  showToast("一键发货完成");
}

async function shipSellerOrder(orderId) {
  await apiFetch(`/api/seller/orders/${orderId}/ship`, { method: "POST" });
  await Promise.all([refreshSellerOrders(), refreshSellerProducts()]);
  showToast("订单已发货");
}

async function delistProduct(productId) {
  if (!window.confirm("确认下架该商品？")) return;
  const output = await apiFetch(`/api/seller/products/${productId}/delist`, { method: "POST" });
  await refreshSellerProducts();
  showToast(`商品状态已变更为 ${PRODUCT_STATUS[output.status] || output.status}`);
}

async function editProduct(productId) {
  const item = state.sellerProducts.find((product) => String(product.productId) === String(productId));
  if (!item) return;
  const productName = window.prompt("商品名", item.productName);
  if (!productName) return;
  const price = window.prompt("价格，单位元", ((item.priceCent || 0) / 100).toFixed(2));
  if (!price) return;
  const status = window.prompt("状态：1 在售，2 预售中，3 正在下架，4 已下架", item.status);
  if (!status) return;
  const description = window.prompt("描述", item.description || "") || "";
  await apiFetch(`/api/seller/products/${productId}`, {
    method: "PUT",
    body: JSON.stringify({
      productName,
      description,
      priceCent: Math.round(Number(price) * 100),
      status: Number(status),
    }),
  });
  await refreshSellerProducts();
  showToast("商品已更新");
}

bootstrapRoleData()
  .catch((err) => {
    showToast(err.message, "error");
    if (String(err.message).includes("登录") || String(err.message).includes("凭证")) {
      localStorage.removeItem("token");
      state.token = "";
    }
  })
  .finally(render);
