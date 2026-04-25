// ITAM Dashboard - Chart.js Configuration

// Chart color scheme
const chartColors = {
    primary: 'rgba(32, 107, 196, 1)',
    primaryFaded: 'rgba(32, 107, 196, 0.2)',
    success: 'rgba(47, 179, 68, 1)',
    successFaded: 'rgba(47, 179, 68, 0.2)',
    purple: 'rgba(174, 62, 201, 1)',
    purpleFaded: 'rgba(174, 62, 201, 0.2)',
    grid: 'rgba(255, 255, 255, 0.1)',
    text: 'rgba(255, 255, 255, 0.7)'
};

// Store chart instances
let cpuChart = null;
let ramChart = null;

// Data storage
const chartData = {
    labels: [],
    cpuValues: [],
    ramValues: [],
    maxDataPoints: 20
};

// Initialize charts when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    initCharts();
    startDataPolling();
});

function initCharts() {
    const chartConfig = {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
            duration: 0 // Disable animation to prevent "flying" points
        },
        interaction: {
            mode: 'index',
            intersect: false,
        },
        plugins: {
            legend: {
                display: false
            },
            tooltip: {
                backgroundColor: 'rgba(22, 27, 34, 0.95)',
                titleColor: '#fff',
                bodyColor: 'rgba(255, 255, 255, 0.8)',
                borderColor: 'rgba(48, 54, 61, 1)',
                borderWidth: 1,
                padding: 12,
                displayColors: false
            }
        },
        scales: {
            x: {
                grid: {
                    color: chartColors.grid,
                    drawBorder: false
                },
                ticks: {
                    color: chartColors.text,
                    maxRotation: 0
                }
            },
            y: {
                min: 0,
                max: 100,
                grid: {
                    color: chartColors.grid,
                    drawBorder: false
                },
                ticks: {
                    color: chartColors.text,
                    callback: function(value) {
                        return value + '%';
                    }
                }
            }
        },
        elements: {
            line: {
                tension: 0 // Straight lines for better stability
            },
            point: {
                radius: 3, // Show points for clarity
                hoverRadius: 6
            }
        }
    };

    // CPU Chart
    const cpuCtx = document.getElementById('cpuChart').getContext('2d');
    cpuChart = new Chart(cpuCtx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'CPU Usage',
                data: [],
                borderColor: chartColors.primary,
                backgroundColor: chartColors.primaryFaded,
                fill: true,
                borderWidth: 2
            }]
        },
        options: chartConfig
    });

    // RAM Chart
    const ramCtx = document.getElementById('ramChart').getContext('2d');
    ramChart = new Chart(ramCtx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'RAM Usage',
                data: [],
                borderColor: chartColors.purple,
                backgroundColor: chartColors.purpleFaded,
                fill: true,
                borderWidth: 2
            }]
        },
        options: chartConfig
    });
}

function updateCharts(stats) {
    if (!stats || !cpuChart || !ramChart) return;

    const now = new Date().toLocaleTimeString('en-US', { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit',
        second: '2-digit'
    });

    // Prevent duplicate timestamps
    if (chartData.labels.length > 0 && chartData.labels[chartData.labels.length - 1] === now) {
        return;
    }

    // Add new data point
    chartData.labels.push(now);
    chartData.cpuValues.push(stats.avgCpu || 0);
    chartData.ramValues.push(stats.avgRam || 0);

    // Keep only last N data points
    if (chartData.labels.length > chartData.maxDataPoints) {
        chartData.labels.shift();
        chartData.cpuValues.shift();
        chartData.ramValues.shift();
    }

    // Update CPU chart
    cpuChart.data.labels = [...chartData.labels];
    cpuChart.data.datasets[0].data = [...chartData.cpuValues];
    cpuChart.update('none');

    // Update RAM chart
    ramChart.data.labels = [...chartData.labels];
    ramChart.data.datasets[0].data = [...chartData.ramValues];
    ramChart.update('none');
}

function startDataPolling() {
    // Poll for chart data every 10 seconds
    setInterval(async () => {
        try {
            const response = await fetch('/api/stats');
            if (response.ok) {
                const stats = await response.json();
                updateCharts(stats);
            }
        } catch (error) {
            console.warn('Failed to fetch stats:', error);
        }
    }, 10000);

    // Initial fetch
    fetch('/api/stats')
        .then(r => r.json())
        .then(updateCharts)
        .catch(() => {});
}

function refreshAll() {
    // Trigger HTMX refresh on all polling elements
    document.querySelectorAll('[hx-get]').forEach(el => {
        htmx.trigger(el, 'htmx:trigger');
    });
    
    // Fetch fresh stats for charts
    fetch('/api/stats')
        .then(r => r.json())
        .then(updateCharts)
        .catch(() => {});
}

// Format bytes to human readable
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Format uptime to human readable
function formatUptime(seconds) {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${mins}m`;
    return `${mins}m`;
}
