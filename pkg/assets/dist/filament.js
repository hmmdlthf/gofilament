/**
 * GoFilament Runtime
 * Minimal Alpine-compatible reactive state & HTMX integration engine.
 */
(function() {
  window.Filament = {
    // Theme Switcher
    theme: {
      get: function() {
        return localStorage.getItem('filament_theme') || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
      },
      set: function(theme) {
        localStorage.setItem('filament_theme', theme);
        if (theme === 'dark') {
          document.documentElement.classList.add('dark');
        } else {
          document.documentElement.classList.remove('dark');
        }
      },
      toggle: function() {
        this.set(this.get() === 'dark' ? 'light' : 'dark');
      },
      init: function() {
        this.set(this.get());
      }
    },

    // Notification System
    notify: function(options) {
      const container = document.getElementById('filament-notifications') || this.createNotificationContainer();
      const toast = document.createElement('div');
      toast.className = 'toast animate-slide-in';
      
      const type = options.type || 'info';
      const color = type === 'success' ? '#10b981' : type === 'danger' ? '#ef4444' : type === 'warning' ? '#f59e0b' : '#3b82f6';
      
      toast.innerHTML = `
        <div style="width: 0.5rem; height: 100%; border-radius: 9999px; background-color: ${color};"></div>
        <div style="flex: 1;">
          <div style="font-weight: 600; font-size: 0.875rem;">${options.title || 'Notification'}</div>
          ${options.body ? `<div style="font-size: 0.75rem; color: #64748b; margin-top: 0.25rem;">${options.body}</div>` : ''}
        </div>
        <button onclick="this.parentElement.remove()" style="background: none; border: none; cursor: pointer; color: #94a3b8;">&times;</button>
      `;

      container.appendChild(toast);
      setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(10px)';
        setTimeout(() => toast.remove(), 250);
      }, options.duration || 4000);
    },

    createNotificationContainer: function() {
      const container = document.createElement('div');
      container.id = 'filament-notifications';
      container.className = 'notifications-container';
      document.body.appendChild(container);
      return container;
    },

    // Modal Manager
    openModal: function(modalId) {
      const modal = document.getElementById(modalId);
      if (modal) {
        modal.style.display = 'flex';
      }
    },

    closeModal: function(modalId) {
      const modal = document.getElementById(modalId);
      if (modal) {
        modal.style.display = 'none';
      }
    },

    // Live Field Debounce Update for Forms
    liveUpdate: function(formId, fieldName, triggerUrl) {
      const form = document.getElementById(formId);
      if (!form) return;
      const formData = new FormData(form);
      formData.set('_live_field', fieldName);

      fetch(triggerUrl || form.action, {
        method: 'POST',
        headers: {
          'X-Filament-Live': 'true',
          'Accept': 'text/html'
        },
        body: formData
      })
      .then(r => r.text())
      .then(html => {
        // Targeted swap of the form or fields
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');
        const newForm = doc.getElementById(formId);
        if (newForm) {
          form.replaceWith(newForm);
        }
      })
      .catch(err => console.error('Filament live update error:', err));
    },

    // Notification Drawer Management
    openNotificationDrawer: function() {
      const d = document.getElementById('filament-notifications-drawer');
      if (d) d.style.display = 'flex';
    },

    closeNotificationDrawer: function() {
      const d = document.getElementById('filament-notifications-drawer');
      if (d) d.style.display = 'none';
    },

    markNotificationRead: function(url, id) {
      fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: id })
      }).then(() => {
        const item = document.getElementById('notif-item-' + id);
        if (item) item.classList.remove('unread');
        this.decrementBadge();
      }).catch(err => console.error('mark notification read error:', err));
    },

    markAllNotificationsRead: function(url) {
      fetch(url, { method: 'POST' }).then(() => {
        document.querySelectorAll('.notification-item.unread').forEach(el => el.classList.remove('unread'));
        const badge = document.getElementById('notif-unread-badge');
        if (badge) badge.style.display = 'none';
        const count = document.getElementById('drawer-unread-count');
        if (count) count.textContent = '0 unread';
      }).catch(err => console.error('mark all read error:', err));
    },

    clearAllNotifications: function(url) {
      fetch(url, { method: 'POST' }).then(() => {
        const list = document.getElementById('notifications-drawer-list');
        if (list) list.innerHTML = '<div style="text-align: center; color: var(--gray-400); padding: 2rem;">No notifications.</div>';
        const badge = document.getElementById('notif-unread-badge');
        if (badge) badge.style.display = 'none';
        const count = document.getElementById('drawer-unread-count');
        if (count) count.textContent = '0 unread';
      }).catch(err => console.error('clear notifications error:', err));
    },

    initSSE: function(streamUrl) {
      if (!window.EventSource || !streamUrl) return;
      try {
        const es = new EventSource(streamUrl);
        es.addEventListener('notification', (e) => {
          try {
            const data = JSON.parse(e.data);
            this.notify(data);
            this.incrementBadge();
            this.appendDrawerItem(data);
          } catch(err) {
            console.error('SSE notification parse error:', err);
          }
        });
      } catch(err) {
        console.error('SSE initialization failed:', err);
      }
    },

    incrementBadge: function() {
      const badge = document.getElementById('notif-unread-badge');
      if (badge) {
        let count = parseInt(badge.textContent || '0', 10) + 1;
        badge.textContent = count;
        badge.style.display = 'inline-flex';
      }
      const count = document.getElementById('drawer-unread-count');
      if (count) {
        let n = parseInt(count.textContent || '0', 10) + 1;
        count.textContent = n + ' unread';
      }
    },

    decrementBadge: function() {
      const badge = document.getElementById('notif-unread-badge');
      if (badge) {
        let count = parseInt(badge.textContent || '1', 10) - 1;
        if (count <= 0) {
          badge.style.display = 'none';
        } else {
          badge.textContent = count;
        }
      }
      const count = document.getElementById('drawer-unread-count');
      if (count) {
        let n = Math.max(0, parseInt(count.textContent || '1', 10) - 1);
        count.textContent = n + ' unread';
      }
    },

    appendDrawerItem: function(data) {
      const list = document.getElementById('notifications-drawer-list');
      if (!list) return;
      const emptyState = list.querySelector('.empty-notif-msg');
      if (emptyState) emptyState.remove();

      const item = document.createElement('div');
      item.id = 'notif-item-' + data.id;
      item.className = 'notification-item unread';
      item.innerHTML = `
        <div style="font-weight: 600; font-size: 0.875rem;">${data.title || 'Notification'}</div>
        ${data.body ? `<div style="font-size: 0.8125rem; color: #64748b; margin-top: 0.25rem;">${data.body}</div>` : ''}
        <div style="font-size: 0.75rem; color: #94a3b8; margin-top: 0.5rem;">Just now</div>
      `;
      list.prepend(item);
    }
  };

  // Initialize theme on load
  document.addEventListener('DOMContentLoaded', () => {
    window.Filament.theme.init();
  });
})();
