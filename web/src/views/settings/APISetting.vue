<template>
  <div class="api-setting-page">
    <div class="setting-section">
      <div class="setting-card">
        <div class="setting-row">
          <span class="setting-label">服务地址</span>
          <span class="setting-value">{{ origin }}</span>
        </div>
        <div class="setting-row">
          <span class="setting-label">版本</span>
          <span class="setting-value">Sub-Store Go</span>
        </div>
      </div>
    </div>
    <div class="setting-section">
      <div class="setting-card">
        <button class="logout-btn" @click="logout">
          <font-awesome-icon icon="fa-solid fa-right-from-bracket" />
          退出登录
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const origin = window.location.origin;

const logout = async () => {
  await fetch('/api/auth/logout', { method: 'POST' });
  window.location.reload();
};
</script>

<style lang="scss" scoped>
.api-setting-page {
  padding: 16px 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.setting-section {}

.setting-card {
  background: var(--card-color);
  border-radius: 12px;
  overflow: hidden;
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  border-bottom: 1px solid rgba(0,0,0,0.04);

  &:last-child { border-bottom: none; }
}

.setting-label {
  font-size: 15px;
  color: var(--primary-text-color);
}

.setting-value {
  font-size: 14px;
  color: var(--comment-text-color);
  font-family: monospace;
}

.logout-btn {
  width: 100%;
  padding: 14px 16px;
  background: none;
  border: none;
  font-size: 15px;
  font-weight: 500;
  color: var(--danger-color, #E56459);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background 0.12s;

  &:hover { background: rgba(229,100,89,0.06); }
}
</style>
