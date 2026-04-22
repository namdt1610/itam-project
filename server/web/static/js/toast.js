// Toast Notification System
const Toast = {
    container: null,
    
    init() {
        if (this.container) return;
        this.container = document.createElement('div');
        this.container.className = 'toast-container';
        document.body.appendChild(this.container);
    },
    
    show(message, type = 'info', duration = 4000) {
        this.init();
        
        const icons = {
            success: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 6L9 17l-5-5"/></svg>',
            error: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M15 9l-6 6M9 9l6 6"/></svg>',
            warning: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 9v4M12 17h.01"/><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/></svg>',
            info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>'
        };
        
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.innerHTML = `
            <span class="toast-icon">${icons[type] || icons.info}</span>
            <span class="toast-message">${message}</span>
            <button class="toast-close" onclick="Toast.dismiss(this.parentElement)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
            </button>
        `;
        
        this.container.appendChild(toast);
        
        if (duration > 0) {
            setTimeout(() => this.dismiss(toast), duration);
        }
        
        return toast;
    },
    
    dismiss(toast) {
        if (!toast || !toast.parentElement) return;
        toast.classList.add('hide');
        setTimeout(() => toast.remove(), 300);
    },
    
    success(message, duration) { return this.show(message, 'success', duration); },
    error(message, duration) { return this.show(message, 'error', duration); },
    warning(message, duration) { return this.show(message, 'warning', duration); },
    info(message, duration) { return this.show(message, 'info', duration); }
};

// HTMX Event Listeners for Toasts
document.addEventListener('htmx:afterRequest', function(event) {
    const xhr = event.detail.xhr;
    if (!xhr) return;
    
    // Success responses
    if (xhr.status >= 200 && xhr.status < 300) {
        // Only show for specific actions
        if (event.detail.pathInfo.requestPath.includes('/resolve')) {
            Toast.success('Alert resolved successfully');
        }
    }
    
    // Error responses
    if (xhr.status >= 400) {
        if (xhr.status === 429) {
            Toast.warning('Server is busy. Please wait...');
        } else if (xhr.status === 401) {
            Toast.error('Unauthorized access');
        } else if (xhr.status >= 500) {
            Toast.error('Server error. Please try again later.');
        }
    }
});

// Network error handler
document.addEventListener('htmx:sendError', function(event) {
    Toast.error('Network error. Check your connection.');
});

// Optional: Show toast on page load
document.addEventListener('DOMContentLoaded', function() {
    // Check connection to server
    fetch('/health')
        .then(r => {
            if (r.ok) Toast.success('Connected to ITAM server', 2000);
        })
        .catch(() => {
            Toast.error('Cannot connect to server');
        });
});
