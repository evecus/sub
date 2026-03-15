<template>
  <div class="login-wrap">
    <div class="login-container">
      <!-- Logo -->
      <div class="login-logo">
        <img src="@/assets/icons/logo-red.png" alt="Sub-Store" class="logo-img" />
        <h1 class="logo-title">Sub-Store</h1>
        <p class="logo-subtitle">请登录以继续</p>
      </div>

      <!-- Form card -->
      <div class="login-card" :style="shake ? { animation: 'shakeX 0.5s ease' } : {}">
        <div class="login-field">
          <label class="login-label">用户名</label>
          <input
            id="login-username"
            v-model="username"
            class="login-input"
            placeholder="admin"
            autocomplete="username"
            @keydown.enter="submit"
          />
        </div>
        <div class="login-field">
          <label class="login-label">密码</label>
          <div class="login-input-wrap">
            <input
              v-model="password"
              class="login-input"
              :type="showPwd ? 'text' : 'password'"
              placeholder="••••••••••••••••"
              autocomplete="current-password"
              @keydown.enter="submit"
            />
            <button class="pwd-toggle" type="button" @click="showPwd = !showPwd">
              <font-awesome-icon :icon="showPwd ? 'fa-solid fa-eye-slash' : 'fa-solid fa-eye'" />
            </button>
          </div>
        </div>

        <div v-if="error" class="login-error">{{ error }}</div>

        <button class="login-btn" :disabled="loading" @click="submit">
          <span v-if="loading" class="btn-spinner"></span>
          <font-awesome-icon v-else icon="fa-solid fa-right-to-bracket" />
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </div>

      <p class="login-hint">默认密码见启动日志 · 通过 <code>--auth</code> 参数自定义</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';

const emit = defineEmits(['login']);

const username = ref('');
const password = ref('');
const showPwd = ref(false);
const loading = ref(false);
const error = ref('');
const shake = ref(false);

onMounted(() => {
  document.getElementById('login-username')?.focus();
});

const submit = async () => {
  if (!username.value || !password.value) {
    error.value = '请输入用户名和密码';
    return;
  }
  loading.value = true;
  error.value = '';
  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    });
    if (res.ok) {
      emit('login');
    } else {
      const j = await res.json();
      error.value = j.error || '用户名或密码错误';
      shake.value = true;
      setTimeout(() => (shake.value = false), 600);
    }
  } catch {
    error.value = '网络错误，请稍后重试';
  } finally {
    loading.value = false;
  }
};
</script>

<style lang="scss" scoped>
.login-wrap {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--background-color);
  padding: 20px;
}

.login-container {
  width: 100%;
  max-width: 360px;
}

.login-logo {
  text-align: center;
  margin-bottom: 28px;
}

.logo-img {
  width: 72px;
  height: 72px;
  border-radius: 18px;
  margin-bottom: 14px;
}

.logo-title {
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.5px;
  color: var(--primary-text-color);
  margin: 0 0 6px;
}

.logo-subtitle {
  font-size: 14px;
  color: var(--comment-text-color);
  margin: 0;
}

.login-card {
  background: var(--card-color);
  border-radius: 18px;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.login-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.login-label {
  font-size: 13px;
  color: var(--second-text-color);
  font-weight: 500;
}

.login-input-wrap {
  position: relative;
}

.login-input {
  width: 100%;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1.5px solid var(--divider-color, rgba(0,0,0,0.08));
  background: var(--background-color);
  font-size: 15px;
  color: var(--primary-text-color);
  outline: none;
  font-family: inherit;
  transition: border-color 0.15s;
  box-sizing: border-box;

  &:focus {
    border-color: var(--primary-color, #478EF2);
  }

  &::placeholder {
    color: var(--lowest-text-color);
  }
}

.pwd-toggle {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  cursor: pointer;
  color: var(--comment-text-color);
  padding: 0;
  display: flex;
  align-items: center;
  font-size: 14px;
}

.login-error {
  background: rgba(229, 100, 89, 0.1);
  border: 1px solid rgba(229, 100, 89, 0.3);
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 13px;
  color: var(--danger-color, #E56459);
}

.login-btn {
  width: 100%;
  padding: 13px;
  border-radius: 10px;
  background: var(--primary-color, #478EF2);
  border: none;
  color: white;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: background 0.15s;
  margin-top: 4px;

  &:hover { background: #3a7ee0; }
  &:disabled { opacity: 0.5; cursor: not-allowed; }
}

.btn-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid rgba(255,255,255,0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  display: inline-block;
}

.login-hint {
  text-align: center;
  margin-top: 16px;
  font-size: 12px;
  color: var(--lowest-text-color);

  code {
    font-family: monospace;
    background: rgba(0,0,0,0.06);
    padding: 1px 4px;
    border-radius: 3px;
  }
}

@keyframes spin { to { transform: rotate(360deg); } }
@keyframes shakeX {
  0%, 100% { transform: translateX(0); }
  15% { transform: translateX(-8px); }
  30% { transform: translateX(8px); }
  45% { transform: translateX(-5px); }
  60% { transform: translateX(5px); }
  90% { transform: translateX(2px); }
}
</style>
