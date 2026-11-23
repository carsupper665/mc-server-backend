<script setup>
import { computed, h, onMounted, ref } from 'vue';
import { RouterLink } from 'vue-router';
import {
  NAlert, NButton, NCard, NDataTable, NForm, NFormItem, NIcon, NInput, NModal, NSelect, NSpace, NTag, NText, useMessage,
} from 'naive-ui';
import { PlusOutlined, ReloadOutlined } from '@vicons/antd';
import api, { getSanitizedErrorMessage } from '../api';
import { useAuthStore } from '../store/auth';
import { roleName, roleOptions } from '../utils/roles';

const message = useMessage();
const authStore = useAuthStore();
const isRoot = computed(() => authStore.isAdmin);

const users = ref([]);
const loading = ref(false);

const MIN_PASSWORD = 8;
const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const assignableRoles = roleOptions();

const formatDate = value => {
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString('zh-TW', { hour12: false });
};

const fetchUsers = async () => {
  if (!isRoot.value) return;
  loading.value = true;
  try {
    const res = await api.get('/op/v1/users', { silent: true });
    users.value = Array.isArray(res?.data) ? res.data : [];
  } catch (err) {
    message.error('無法載入使用者列表: ' + getSanitizedErrorMessage(err));
  } finally {
    loading.value = false;
  }
};

const isSelf = user => String(user.id) === String(authStore.user?.userId || '') || (!authStore.user?.userId && user.username === authStore.user?.username);

// ---------------- 建立使用者 ----------------
const emptyCreateForm = () => ({ username: '', display_name: '', email: '', password: '', role: 1 });
const showCreate = ref(false);
const createForm = ref(emptyCreateForm());
const creating = ref(false);

// 與後端 RegisterUserRequest 的 binding 一致
const createError = computed(() => {
  const f = createForm.value;
  if (!f.username.trim()) return '請輸入帳號';
  if (!f.display_name.trim()) return '請輸入顯示名稱';
  if (!emailPattern.test(f.email.trim())) return '請輸入有效的 email';
  if (f.password.length < MIN_PASSWORD) return `密碼至少 ${MIN_PASSWORD} 碼`;
  if (!assignableRoles.some(r => r.value === f.role)) return '請選擇角色';
  return '';
});

const openCreate = () => {
  createForm.value = emptyCreateForm();
  showCreate.value = true;
};

const submitCreate = async () => {
  if (creating.value) return;
  if (createError.value) {
    message.warning(createError.value);
    return;
  }
  creating.value = true;
  try {
    const f = createForm.value;
    await api.post('/op/v1/register', {
      username: f.username.trim(),
      display_name: f.display_name.trim(),
      email: f.email.trim(),
      password: f.password,
      role: f.role,
    }, { silent: true });
    message.success('使用者已建立');
    showCreate.value = false;
    await fetchUsers();
  } catch (err) {
    message.error('建立失敗: ' + getSanitizedErrorMessage(err));
  } finally {
    creating.value = false;
  }
};

// ---------------- 編輯使用者 ----------------
const showEdit = ref(false);
const editing = ref(null);
const editForm = ref({ username: '', display_name: '', email: '', role: null });
const saving = ref(false);
const confirmSelfRole = ref(false);

const openEdit = user => {
  editing.value = user;
  editForm.value = { username: user.username || '', display_name: user.display_name || '', email: user.email || '', role: user.role };
  showEdit.value = true;
};

// 只送有變更的欄位；FGF 帳號不送角色與顯示名稱
const editChanges = computed(() => {
  const user = editing.value;
  if (!user) return {};
  const f = editForm.value;
  const changes = {};
  if (f.username.trim() && f.username.trim() !== user.username) changes.new_username = f.username.trim();
  if (f.email.trim() && f.email.trim() !== user.email) changes.new_email = f.email.trim();
  if (!user.is_fgf) {
    if (f.display_name.trim() && f.display_name.trim() !== user.display_name) changes.new_display_name = f.display_name.trim();
    if (f.role !== user.role && f.role != null) changes.new_role = f.role;
  }
  return changes;
});

const editError = computed(() => {
  const f = editForm.value;
  if (!f.username.trim()) return '帳號不能空白';
  if (!emailPattern.test(f.email.trim())) return '請輸入有效的 email';
  if (!editing.value?.is_fgf && !f.display_name.trim()) return '顯示名稱不能空白';
  return '';
});

const sendEdit = async () => {
  saving.value = true;
  try {
    await api.post('/op/v1/edit', { user_id: editing.value.id, ...editChanges.value }, { silent: true });
    message.success('使用者已更新');
    showEdit.value = false;
    confirmSelfRole.value = false;
    await fetchUsers();
    if (isSelf(editing.value)) await authStore.fetchUser().catch(() => {});
  } catch (err) {
    message.error('更新失敗: ' + getSanitizedErrorMessage(err));
  } finally {
    saving.value = false;
  }
};

const submitEdit = async () => {
  if (saving.value) return;
  if (editError.value) {
    message.warning(editError.value);
    return;
  }
  if (Object.keys(editChanges.value).length === 0) {
    message.info('沒有變更');
    return;
  }
  // Root 改自己的角色要再確認一次，避免把自己降級
  if (isSelf(editing.value) && 'new_role' in editChanges.value) {
    confirmSelfRole.value = true;
    return;
  }
  await sendEdit();
};

// ---------------- 重設密碼 ----------------
const showReset = ref(false);
const resetting = ref(null);
const resetForm = ref({ password: '', confirm: '' });
const resetBusy = ref(false);

const resetError = computed(() => {
  const f = resetForm.value;
  if (f.password.length < MIN_PASSWORD) return `密碼至少 ${MIN_PASSWORD} 碼`;
  if (f.password !== f.confirm) return '兩次輸入的密碼不一致';
  return '';
});

const openReset = user => {
  resetting.value = user;
  resetForm.value = { password: '', confirm: '' };
  showReset.value = true;
};

const submitReset = async () => {
  if (resetBusy.value) return;
  if (resetError.value) {
    message.warning(resetError.value);
    return;
  }
  resetBusy.value = true;
  try {
    await api.post('/op/v1/reset', { user_id: resetting.value.id, new_password: resetForm.value.password }, { silent: true });
    message.success('密碼已重設');
    showReset.value = false;
  } catch (err) {
    message.error('重設失敗: ' + getSanitizedErrorMessage(err));
  } finally {
    resetBusy.value = false;
  }
};

const columns = [
  { title: 'ID', key: 'id', width: 70 },
  { title: '帳號', key: 'username' },
  { title: '顯示名稱', key: 'display_name' },
  { title: 'Email', key: 'email' },
  { title: '角色', key: 'role', render: row => roleName(row.role) },
  {
    title: 'FGF',
    key: 'is_fgf',
    render: row => h(NTag, { size: 'small', type: row.is_fgf ? 'info' : 'default' }, { default: () => (row.is_fgf ? 'FGF' : '本地') }),
  },
  { title: '建立時間', key: 'created_at', render: row => formatDate(row.created_at) },
  {
    title: '操作',
    key: 'actions',
    render: row => h(NSpace, { size: 8 }, {
      default: () => [
        h(NButton, { size: 'small', ghost: true, type: 'primary', class: 'edit-user-btn', onClick: () => openEdit(row) }, { default: () => '編輯' }),
        // FGF 帳號不用密碼登入
        row.is_fgf ? null : h(NButton, { size: 'small', ghost: true, type: 'warning', class: 'reset-user-btn', onClick: () => openReset(row) }, { default: () => '重設密碼' }),
      ].filter(Boolean),
    }),
  },
];

onMounted(fetchUsers);
</script>

<template>
  <div class="admin-users-view">
    <n-space v-if="!isRoot" vertical :size="20" role="status">
      <n-text>沒有權限查看此頁面</n-text>
      <RouterLink to="/">
        <n-button>返回 Dashboard</n-button>
      </RouterLink>
    </n-space>

    <n-space v-else vertical :size="24">
      <n-space justify="space-between" align="center">
        <div>
          <h2 class="title">USER MANAGEMENT</h2>
          <n-text depth="3">ROOT ONLY</n-text>
        </div>
        <n-space>
          <n-button ghost :loading="loading" @click="fetchUsers">
            <template #icon><n-icon><ReloadOutlined /></n-icon></template>
            REFRESH
          </n-button>
          <n-button type="primary" class="create-user-btn" @click="openCreate">
            <template #icon><n-icon><PlusOutlined /></n-icon></template>
            新增使用者
          </n-button>
        </n-space>
      </n-space>

      <n-card class="table-card" :bordered="false" content-style="padding: 0;">
        <div class="table-container">
          <!-- scroll-x：窄螢幕時表格左右捲動，不把欄位擠成一字一行 -->
          <n-data-table :columns="columns" :data="users" :loading="loading" :bordered="false" :row-key="row => row.id" :scroll-x="900" />
        </div>
      </n-card>
    </n-space>

    <n-modal v-model:show="showCreate" preset="card" title="新增使用者" :style="{ width: 'min(480px, 94vw)' }" :closable="!creating" :mask-closable="!creating">
      <n-form label-placement="top">
        <n-form-item label="帳號"><n-input v-model:value="createForm.username" placeholder="username" /></n-form-item>
        <n-form-item label="顯示名稱"><n-input v-model:value="createForm.display_name" /></n-form-item>
        <n-form-item label="Email"><n-input v-model:value="createForm.email" placeholder="user@example.com" /></n-form-item>
        <n-form-item :label="`密碼（至少 ${MIN_PASSWORD} 碼）`">
          <n-input v-model:value="createForm.password" type="password" show-password-on="click" />
        </n-form-item>
        <n-form-item label="角色"><n-select v-model:value="createForm.role" :options="assignableRoles" /></n-form-item>
      </n-form>
      <n-text v-if="createError" depth="3" class="form-hint">{{ createError }}</n-text>
      <template #footer>
        <n-space justify="end">
          <n-button :disabled="creating" @click="showCreate = false">取消</n-button>
          <n-button type="primary" class="submit-create-btn" :loading="creating" :disabled="!!createError" @click="submitCreate">建立</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal v-model:show="showEdit" preset="card" title="編輯使用者" :style="{ width: 'min(480px, 94vw)' }" :closable="!saving" :mask-closable="!saving">
      <n-form v-if="editing" label-placement="top">
        <n-form-item label="帳號"><n-input v-model:value="editForm.username" /></n-form-item>
        <n-form-item label="顯示名稱"><n-input v-model:value="editForm.display_name" :disabled="editing.is_fgf" /></n-form-item>
        <n-form-item label="Email"><n-input v-model:value="editForm.email" /></n-form-item>
        <n-form-item label="角色">
          <n-select v-model:value="editForm.role" :options="assignableRoles" :disabled="editing.is_fgf" :placeholder="roleName(editing.role)" />
        </n-form-item>
        <n-alert v-if="editing.is_fgf" type="info" class="fgf-note">角色與顯示名稱由 FGF 同步，無法在這裡修改。</n-alert>
      </n-form>
      <n-text v-if="editError" depth="3" class="form-hint">{{ editError }}</n-text>
      <template #footer>
        <n-space justify="end">
          <n-button :disabled="saving" @click="showEdit = false">取消</n-button>
          <n-button type="primary" class="submit-edit-btn" :loading="saving" :disabled="!!editError" @click="submitEdit">儲存</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal
      v-model:show="confirmSelfRole"
      preset="dialog"
      type="warning"
      title="確定要修改自己的角色？"
      positive-text="確認修改"
      negative-text="取消"
      :loading="saving"
      @positive-click="sendEdit"
      @negative-click="confirmSelfRole = false"
    >
      你正在把自己的角色改成 {{ roleName(editForm.role) }}。降級後可能無法再進入使用者管理頁。
    </n-modal>

    <n-modal v-model:show="showReset" preset="card" title="重設密碼" :style="{ width: 'min(420px, 94vw)' }" :closable="!resetBusy" :mask-closable="!resetBusy">
      <n-text v-if="resetting" depth="3">帳號：{{ resetting.username }}</n-text>
      <n-form label-placement="top" class="reset-form">
        <n-form-item :label="`新密碼（至少 ${MIN_PASSWORD} 碼）`">
          <n-input v-model:value="resetForm.password" type="password" show-password-on="click" />
        </n-form-item>
        <n-form-item label="再輸入一次">
          <n-input v-model:value="resetForm.confirm" type="password" show-password-on="click" />
        </n-form-item>
      </n-form>
      <n-text v-if="resetError" depth="3" class="form-hint">{{ resetError }}</n-text>
      <template #footer>
        <n-space justify="end">
          <n-button :disabled="resetBusy" @click="showReset = false">取消</n-button>
          <n-button type="warning" class="submit-reset-btn" :loading="resetBusy" :disabled="!!resetError" @click="submitReset">重設</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.title {
  margin: 0;
  font-size: 24px;
  font-family: 'Fira Code', monospace;
}

.table-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
  overflow: hidden;
}

.table-container {
  overflow-x: auto;
}

.form-hint {
  font-size: 12px;
}

.fgf-note {
  margin-top: 4px;
}

.reset-form {
  margin-top: 12px;
}
</style>
