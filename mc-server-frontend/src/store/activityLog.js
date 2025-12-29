import { defineStore } from 'pinia';

/**
 * 本地活動紀錄 Store
 *
 * 彌補後端沒有 Audit Log 的缺憾，
 * 讓使用者可以查看自己最近的操作歷史。
 */

const STORAGE_KEY = 'mc_activity_log';
const MAX_RECORDS = 100; // 增加歷史記錄數量

export const useActivityLogStore = defineStore('activityLog', {
  state: () => ({
    activities: [],
  }),

  getters: {
    /**
     * 取得最近 N 筆活動記錄
     */
    recentActivities:
      state =>
        (count = 10) => {
          return state.activities.slice(0, count);
        },

    /**
     * 取得今日活動記錄
     */
    todayActivities: state => {
      const today = new Date().toDateString();
      return state.activities.filter(a => new Date(a.timestamp).toDateString() === today);
    },

    /**
     * 取得特定類型的活動
     */
    activitiesByType: state => actionType => {
      return state.activities.filter(a =>
        a.action.toLowerCase().includes(actionType.toLowerCase())
      );
    },

    /**
     * 取得活動統計
     */
    stats: state => {
      const successCount = state.activities.filter(a => a.status === 'success').length;
      const errorCount = state.activities.filter(a => a.status === 'error').length;
      return {
        total: state.activities.length,
        success: successCount,
        error: errorCount,
        successRate:
          state.activities.length > 0
            ? Math.round((successCount / state.activities.length) * 100)
            : 100,
      };
    },
  },

  actions: {
    /**
     * 初始化：從 LocalStorage 載入
     */
    init() {
      try {
        const stored = localStorage.getItem(STORAGE_KEY);
        if (stored) {
          this.activities = JSON.parse(stored);

        }
      } catch (err) {
        console.error('[ActivityLog] Failed to load from storage:', err);
        this.activities = [];
      }
    },

    /**
     * 記錄活動
     * @param {Object} activity - 活動物件
     * @param {string} activity.action - 操作類型 (e.g., "Start Server")
     * @param {string} activity.target - 操作目標 (e.g., "mcsvv-xxx")
     * @param {string} activity.targetName - 目標顯示名稱 (e.g., "My Server")
     * @param {'success'|'error'} activity.status - 操作結果
     * @param {string} [activity.details] - 額外詳情
     * @param {string} [activity.category] - 分類 (server/amongus/macro/system)
     */
    log(activity) {
      const record = {
        id: Date.now().toString(36) + Math.random().toString(36).substr(2, 5),
        timestamp: Date.now(),
        action: activity.action,
        target: activity.target || '',
        targetName: activity.targetName || activity.target || '',
        status: activity.status || 'success',
        details: activity.details || '',
        category: activity.category || 'server',
      };

      // 加入到陣列開頭
      this.activities.unshift(record);

      // 限制記錄數量
      if (this.activities.length > MAX_RECORDS) {
        this.activities = this.activities.slice(0, MAX_RECORDS);
      }

      // 儲存到 LocalStorage
      this.save();
    },

    /**
     * 記錄伺服器啟動
     */
    logServerStart(serverId, serverName) {
      this.log({
        action: '啟動伺服器',
        target: serverId,
        targetName: serverName || serverId,
        status: 'success',
        category: 'server',
      });
    },

    /**
     * 記錄伺服器停止
     */
    logServerStop(serverId, serverName) {
      this.log({
        action: '停止伺服器',
        target: serverId,
        targetName: serverName || serverId,
        status: 'success',
        category: 'server',
      });
    },

    /**
     * 記錄設定變更
     */
    logPropertyChange(serverId, serverName) {
      this.log({
        action: '修改伺服器設定',
        target: serverId,
        targetName: serverName || serverId,
        status: 'success',
        category: 'server',
      });
    },

    /**
     * 記錄備份還原
     */
    logRecover(serverId, serverName, backupName) {
      this.log({
        action: '還原備份',
        target: serverId,
        targetName: serverName || serverId,
        status: 'success',
        details: `備份檔案: ${backupName}`,
        category: 'server',
      });
    },

    /**
     * 記錄備份建立
     */
    logBackup(serverId, serverName) {
      this.log({
        action: '建立備份',
        target: serverId,
        targetName: serverName || serverId,
        status: 'success',
        category: 'server',
      });
    },

    /**
     * 記錄伺服器建立
     */
    logServerCreate(serverId, serverName) {
      this.log({
        action: '建立伺服器',
        target: serverId,
        targetName: serverName || serverId,
        status: 'success',
        category: 'server',
      });
    },

    /**
     * 記錄登入
     */
    logLogin(username) {
      this.log({
        action: '登入系統',
        target: username,
        targetName: username,
        status: 'success',
        category: 'system',
      });
    },

    /**
     * 記錄錯誤操作
     */
    logError(action, target, errorMessage) {
      this.log({
        action,
        target,
        targetName: target,
        status: 'error',
        details: errorMessage,
        category: 'system',
      });
    },

    // ===== Among Us 相關 =====

    /**
     * 記錄 Among Us 房間建立
     */
    logAmongUsCreate(roomId) {
      this.log({
        action: '建立 Among Us 房間',
        target: roomId,
        targetName: `Room #${roomId?.substring(0, 6) || 'Unknown'}`,
        status: 'success',
        category: 'amongus',
      });
    },

    /**
     * 記錄 Among Us 遊戲結束
     */
    logAmongUsEnd(roomId) {
      this.log({
        action: '結束 Among Us 遊戲',
        target: roomId,
        targetName: `Room #${roomId?.substring(0, 6) || 'Unknown'}`,
        status: 'success',
        category: 'amongus',
      });
    },

    // ===== Macro 相關 =====

    /**
     * 記錄巨集執行
     */
    logMacroExecute(macroName, serverId, serverName) {
      this.log({
        action: '執行巨集',
        target: serverId,
        targetName: serverName || serverId,
        status: 'success',
        details: `巨集: ${macroName}`,
        category: 'macro',
      });
    },

    /**
     * 記錄巨集建立/修改
     */
    logMacroSave(macroName) {
      this.log({
        action: '儲存巨集',
        target: macroName,
        targetName: macroName,
        status: 'success',
        category: 'macro',
      });
    },

    /**
     * 記錄巨集刪除
     */
    logMacroDelete(macroName) {
      this.log({
        action: '刪除巨集',
        target: macroName,
        targetName: macroName,
        status: 'success',
        category: 'macro',
      });
    },

    /**
     * 儲存到 LocalStorage
     */
    save() {
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(this.activities));
      } catch (err) {
        console.error('[ActivityLog] Failed to save to storage:', err);
      }
    },

    /**
     * 清除所有記錄
     */
    clear() {
      this.activities = [];
      localStorage.removeItem(STORAGE_KEY);
    },

    /**
     * 匯出記錄為 JSON
     */
    exportAsJson() {
      return JSON.stringify(this.activities, null, 2);
    },
  },
});

/**
 * 格式化相對時間
 * @param {number} timestamp - 時間戳記
 * @returns {string} - 相對時間字串
 */
export function formatRelativeTime(timestamp) {
  const now = Date.now();
  const diff = now - timestamp;

  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 10) return '剛剛';
  if (seconds < 60) return `${seconds} 秒前`;
  if (minutes < 60) return `${minutes} 分鐘前`;
  if (hours < 24) return `${hours} 小時前`;
  if (days < 7) return `${days} 天前`;

  // 超過一週顯示日期
  const date = new Date(timestamp);
  return `${date.getMonth() + 1}/${date.getDate()}`;
}
