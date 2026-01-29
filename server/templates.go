package main

import (
	"fmt"
	"net/http"
	"sort"
	"time"
)

// renderAssetsTableWithPagination renders the HTMX partial for assets table with pagination
func renderAssetsTableWithPagination(w http.ResponseWriter, query string, page, limit int, sortBy, sortOrder string) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	assets, total, err := store.Search(query, limit, offset, sortBy, sortOrder)
	if err != nil {
		LogError("Failed to search assets: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if total == 0 {
		fmt.Fprintf(w, `
		<table class="table table-vcenter card-table">
			<tbody>
				<tr>
					<td colspan="9" class="text-center text-muted py-4">
						<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-search me-2" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
							<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
							<circle cx="10" cy="10" r="7"/>
							<line x1="21" y1="21" x2="15" y2="15"/>
						</svg>
						No assets found matching "%s"
					</td>
				</tr>
			</tbody>
		</table>`, query)
		return
	}

	// Calculate pagination info
	start := offset + 1
	end := offset + len(assets)
	totalPages := (total + limit - 1) / limit

	// Helper for header sort link
	sortLink := func(col string) string {
		newOrder := "ASC"
		if sortBy == col && sortOrder == "ASC" {
			newOrder = "DESC"
		}
		return fmt.Sprintf("/api/assets/table?q=%s&page=1&limit=%d&sort=%s&order=%s", query, limit, col, newOrder)
	}

	// Helper for sort icon
	sortIcon := func(col string) string {
		if sortBy != col {
			return ""
		}
		if sortOrder == "ASC" {
			// Arrow Up
			return `<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-sm" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><line x1="12" y1="5" x2="12" y2="19"/><line x1="16" y1="9" x2="12" y2="5"/><line x1="8" y1="9" x2="12" y2="5"/></svg>`
		}
		// Arrow Down
		return `<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-sm" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><line x1="12" y1="5" x2="12" y2="19"/><line x1="16" y1="15" x2="12" y2="19"/><line x1="8" y1="15" x2="12" y2="19"/></svg>`
	}

	fmt.Fprintf(w, `
	<table class="table table-vcenter card-table table-hover">
		<thead>
			<tr>
				<th style="width: 1%%">Status</th>
				<th style="width: 200px"><a href="#" class="text-reset text-decoration-none d-block text-truncate" hx-get="%s" hx-target="#assets-table-container" hx-swap="innerHTML">Hostname %s</a></th>
				<th style="width: 120px"><a href="#" class="text-reset text-decoration-none d-block text-truncate" hx-get="%s" hx-target="#assets-table-container" hx-swap="innerHTML">OS %s</a></th>
				<th style="width: 140px"><a href="#" class="text-reset text-decoration-none d-block text-truncate" hx-get="%s" hx-target="#assets-table-container" hx-swap="innerHTML">IP Address %s</a></th>
				<th style="width: 120px"><a href="#" class="text-reset text-decoration-none d-block text-truncate" hx-get="%s" hx-target="#assets-table-container" hx-swap="innerHTML">CPU %s</a></th>
				<th style="width: 120px"><a href="#" class="text-reset text-decoration-none d-block text-truncate" hx-get="%s" hx-target="#assets-table-container" hx-swap="innerHTML">RAM %s</a></th>
				<th style="width: 120px"><a href="#" class="text-reset text-decoration-none d-block text-truncate" hx-get="%s" hx-target="#assets-table-container" hx-swap="innerHTML">Disk %s</a></th>
				<th style="width: 100px"><a href="#" class="text-reset text-decoration-none d-block text-truncate" hx-get="%s" hx-target="#assets-table-container" hx-swap="innerHTML">Uptime %s</a></th>
				<th style="width: 120px"><a href="#" class="text-reset text-decoration-none d-block text-truncate" hx-get="%s" hx-target="#assets-table-container" hx-swap="innerHTML">Last Seen %s</a></th>
			</tr>
		</thead>
		<tbody>`,
		sortLink("hostname"), sortIcon("hostname"),
		sortLink("os"), sortIcon("os"),
		sortLink("ip_lan"), sortIcon("ip_lan"),
		sortLink("cpu_usage"), sortIcon("cpu_usage"),
		sortLink("ram_percent"), sortIcon("ram_percent"),
		sortLink("disk_percent"), sortIcon("disk_percent"),
		sortLink("uptime"), sortIcon("uptime"),
		sortLink("last_seen"), sortIcon("last_seen"))

	for _, asset := range assets {
		isOnline := time.Since(asset.LastSeen) < 5*time.Minute
		statusClass := "status-offline"
		statusText := "Offline"
		if isOnline {
			statusClass = "status-online"
			statusText = "Online"
		}

		lastSeen := formatTimeAgo(asset.LastSeen)
		uptime := formatUptime(asset.Report.Uptime)
		cpuColor := getProgressColor(asset.Report.CPUUsage)
		ramColor := getProgressColor(asset.Report.RAMPercent)
		diskColor := getProgressColor(asset.Report.DiskPercent)

		fmt.Fprintf(w, `
		<tr style="cursor: pointer;" hx-get="/api/assets/detail?mac=%s" hx-target="#modal-content" hx-trigger="click" data-bs-toggle="modal" data-bs-target="#assetModal">
			<td>
				<span class="status-dot %s" title="%s"></span>
				<span class="text-secondary">%s</span>
			</td>
			<td class="text-truncate" style="max-width: 200px;"><strong>%s</strong></td>
			<td class="text-secondary text-truncate" style="max-width: 120px;">%s</td>
			<td class="text-truncate" style="max-width: 140px;"><code>%s</code></td>
			<td style="min-width: 100px;">
				<div class="d-flex align-items-center">
					<span class="me-2">%.1f%%</span>
					<div class="progress progress-thin flex-fill">
						<div class="progress-bar bg-%s" style="width: %.1f%%"></div>
					</div>
				</div>
			</td>
			<td style="min-width: 100px;">
				<div class="d-flex align-items-center">
					<span class="me-2">%.1f%%</span>
					<div class="progress progress-thin flex-fill">
						<div class="progress-bar bg-%s" style="width: %.1f%%"></div>
					</div>
				</div>
			</td>
			<td style="min-width: 100px;">
				<div class="d-flex align-items-center">
					<span class="me-2">%.1f%%</span>
					<div class="progress progress-thin flex-fill">
						<div class="progress-bar bg-%s" style="width: %.1f%%"></div>
					</div>
				</div>
			</td>
			<td class="text-nowrap"><span class="badge bg-blue-lt">%s</span></td>
			<td class="text-secondary text-truncate" style="max-width: 120px;">%s</td>
		</tr>`,
			asset.Report.MACAddress,
			statusClass, statusText, statusText,
			asset.Report.Hostname,
			asset.Report.OS,
			asset.Report.IPLAN,
			asset.Report.CPUUsage, cpuColor, asset.Report.CPUUsage,
			asset.Report.RAMPercent, ramColor, asset.Report.RAMPercent,
			asset.Report.DiskPercent, diskColor, asset.Report.DiskPercent,
			uptime,
			lastSeen)
	}

	fmt.Fprint(w, `
		</tbody>
	</table>`)

	// Pagination Controls
	fmt.Fprintf(w, `
	<div class="card-footer d-flex align-items-center">
		<p class="m-0 text-secondary">Showing <span>%d</span> to <span>%d</span> of <span>%d</span> assets</p>
		<ul class="pagination m-0 ms-auto">`, start, end, total)

	// Previous Button
	if page > 1 {
		fmt.Fprintf(w, `
		<li class="page-item">
			<a class="page-link" href="#" 
				hx-get="/api/assets/table?q=%s&page=%d&limit=%d&sort=%s&order=%s" 
				hx-target="#assets-table-container"
				hx-swap="innerHTML">
				<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
					<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
					<polyline points="15 6 9 12 15 18" />
				</svg>
				Previous
			</a>
		</li>`, query, page-1, limit, sortBy, sortOrder)
	} else {
		fmt.Fprint(w, `
		<li class="page-item disabled">
			<a class="page-link" href="#" tabindex="-1" aria-disabled="true">
				<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
					<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
					<polyline points="15 6 9 12 15 18" />
				</svg>
				Previous
			</a>
		</li>`)
	}

	// Next Button
	if page < totalPages {
		fmt.Fprintf(w, `
		<li class="page-item">
			<a class="page-link" href="#" 
				hx-get="/api/assets/table?q=%s&page=%d&limit=%d&sort=%s&order=%s" 
				hx-target="#assets-table-container"
				hx-swap="innerHTML">
				Next
				<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
					<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
					<polyline points="9 6 15 12 9 18" />
				</svg>
			</a>
		</li>`, query, page+1, limit, sortBy, sortOrder)
	} else {
		fmt.Fprint(w, `
		<li class="page-item disabled">
			<a class="page-link" href="#" tabindex="-1" aria-disabled="true">
				Next
				<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
					<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
					<polyline points="9 6 15 12 9 18" />
				</svg>
			</a>
		</li>`)
	}

	fmt.Fprint(w, `
		</ul>
	</div>`)
}

// renderStatsCards renders the HTMX partial for stats cards
func renderStatsCards(w http.ResponseWriter, total, online int, avgCPU, avgRAM, avgDisk float64) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
	<div class="col-6 col-lg">
		<div class="card card-sm">
			<div class="card-body">
				<div class="row align-items-center">
					<div class="col-auto">
						<span class="bg-primary text-white avatar">
							<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
								<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
								<rect x="3" y="4" width="18" height="12" rx="1"/>
								<line x1="7" y1="20" x2="17" y2="20"/>
								<line x1="9" y1="16" x2="9" y2="20"/>
								<line x1="15" y1="16" x2="15" y2="20"/>
							</svg>
						</span>
					</div>
					<div class="col">
						<div class="font-weight-medium">%d Assets</div>
						<div class="text-secondary">Total devices</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	<div class="col-6 col-lg">
		<div class="card card-sm">
			<div class="card-body">
				<div class="row align-items-center">
					<div class="col-auto">
						<span class="bg-success text-white avatar">
							<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
								<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
								<path d="M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0 -18 0"/>
								<path d="M12 12m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"/>
							</svg>
						</span>
					</div>
					<div class="col">
						<div class="font-weight-medium">%d Online</div>
						<div class="text-secondary">Active now</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	<div class="col-6 col-lg">
		<div class="card card-sm">
			<div class="card-body">
				<div class="row align-items-center">
					<div class="col-auto">
						<span class="bg-azure text-white avatar">
							<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
								<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
								<path d="M3 12a9 9 0 1 0 18 0a9 9 0 0 0 -18 0"/>
								<path d="M12 7v5l3 3"/>
							</svg>
						</span>
					</div>
					<div class="col">
						<div class="font-weight-medium">%.1f%%</div>
						<div class="text-secondary">Avg CPU</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	<div class="col-6 col-lg">
		<div class="card card-sm">
			<div class="card-body">
				<div class="row align-items-center">
					<div class="col-auto">
						<span class="bg-purple text-white avatar">
							<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
								<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
								<path d="M6 3h12"/>
								<path d="M6 3v10a6 6 0 0 0 12 0v-10"/>
								<path d="M6 21h12"/>
							</svg>
						</span>
					</div>
					<div class="col">
						<div class="font-weight-medium">%.1f%%</div>
						<div class="text-secondary">Avg RAM</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	<div class="col-6 col-lg">
		<div class="card card-sm">
			<div class="card-body">
				<div class="row align-items-center">
					<div class="col-auto">
						<span class="bg-orange text-white avatar">
							<svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
								<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
								<path d="M6 5h12a2 2 0 0 1 2 2v10a2 2 0 0 1 -2 2h-12a2 2 0 0 1 -2 -2v-10a2 2 0 0 1 2 -2"/>
								<path d="M8 5v-1a1 1 0 0 1 1 -1h6a1 1 0 0 1 1 1v1"/>
								<path d="M12 14m-2 0a2 2 0 1 0 4 0a2 2 0 1 0 -4 0"/>
							</svg>
						</span>
					</div>
					<div class="col">
						<div class="font-weight-medium">%.1f%%</div>
						<div class="text-secondary">Avg Disk</div>
					</div>
				</div>
			</div>
		</div>
	</div>`, total, online, avgCPU, avgRAM, avgDisk)
}

// renderAssetDetail renders the HTMX partial for asset detail modal
func renderAssetDetail(w http.ResponseWriter, asset *Asset) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	isOnline := time.Since(asset.LastSeen) < 5*time.Minute
	statusBadge := `<span class="badge bg-danger">Offline</span>`
	if isOnline {
		statusBadge = `<span class="badge bg-success">Online</span>`
	}

	uptime := formatUptime(asset.Report.Uptime)
	lastSeen := asset.LastSeen.Format("2006-01-02 15:04:05")
	ramTotal := formatBytes(asset.Report.RAM)
	ramUsed := formatBytes(asset.Report.RAMUsage)
	diskTotal := formatBytes(asset.Report.Disk)
	diskUsed := formatBytes(asset.Report.DiskUsage)

	fmt.Fprintf(w, `
	<div class="modal-header">
		<h5 class="modal-title">%s %s</h5>
		<button type="button" class="btn-close" data-bs-dismiss="modal"></button>
	</div>
	<div class="modal-body">
		<div class="row g-3">
			<div class="col-12">
				<h4 class="mb-3">System Information</h4>
			</div>
			<div class="col-md-6">
				<div class="datagrid">
					<div class="datagrid-item">
						<div class="datagrid-title">Hostname</div>
						<div class="datagrid-content"><strong>%s</strong></div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">Operating System</div>
						<div class="datagrid-content">%s</div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">IP Address</div>
						<div class="datagrid-content"><code>%s</code></div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">MAC Address</div>
						<div class="datagrid-content"><code>%s</code></div>
					</div>
				</div>
			</div>
			<div class="col-md-6">
				<div class="datagrid">
					<div class="datagrid-item">
						<div class="datagrid-title">Serial Number</div>
						<div class="datagrid-content"><code>%s</code></div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">BIOS Version</div>
						<div class="datagrid-content">%s</div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">Uptime</div>
						<div class="datagrid-content">%s</div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">Last Seen</div>
						<div class="datagrid-content">%s</div>
					</div>
				</div>
			</div>
			<div class="col-12 mt-4">
				<h4 class="mb-3">Hardware</h4>
			</div>
			<div class="col-md-6">
				<div class="datagrid">
					<div class="datagrid-item">
						<div class="datagrid-title">CPU</div>
						<div class="datagrid-content">%s (%d cores)</div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">CPU Usage</div>
						<div class="datagrid-content">
							<div class="progress mt-1" style="height: 8px;">
								<div class="progress-bar bg-%s" style="width: %.1f%%"></div>
							</div>
							<small class="text-muted">%.1f%% used</small>
						</div>
					</div>
				</div>
			</div>
			<div class="col-md-6">
				<div class="datagrid">
					<div class="datagrid-item">
						<div class="datagrid-title">RAM</div>
						<div class="datagrid-content">%s / %s</div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">RAM Usage</div>
						<div class="datagrid-content">
							<div class="progress mt-1" style="height: 8px;">
								<div class="progress-bar bg-%s" style="width: %.1f%%"></div>
							</div>
							<small class="text-muted">%.1f%% used</small>
						</div>
					</div>
				</div>
			</div>
			<div class="col-md-6">
				<div class="datagrid">
					<div class="datagrid-item">
						<div class="datagrid-title">Disk</div>
						<div class="datagrid-content">%s / %s</div>
					</div>
					<div class="datagrid-item">
						<div class="datagrid-title">Disk Usage</div>
						<div class="datagrid-content">
							<div class="progress mt-1" style="height: 8px;">
								<div class="progress-bar bg-%s" style="width: %.1f%%"></div>
							</div>
							<small class="text-muted">%.1f%% used</small>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
	<div class="modal-footer">
		<button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
	</div>`,
		asset.Report.Hostname, statusBadge,
		asset.Report.Hostname,
		asset.Report.OS,
		asset.Report.IPLAN,
		asset.Report.MACAddress,
		asset.Report.SerialNumber,
		asset.Report.BiosVersion,
		uptime,
		lastSeen,
		asset.Report.CPU, asset.Report.Cores,
		getProgressColor(asset.Report.CPUUsage), asset.Report.CPUUsage, asset.Report.CPUUsage,
		ramUsed, ramTotal,
		getProgressColor(asset.Report.RAMPercent), asset.Report.RAMPercent, asset.Report.RAMPercent,
		diskUsed, diskTotal,
		getProgressColor(asset.Report.DiskPercent), asset.Report.DiskPercent, asset.Report.DiskPercent,
	)
}

// renderProcessesTable renders the HTMX partial for processes table
func renderProcessesTable(w http.ResponseWriter) {
	allProcesses := store.AllProcesses()

	// Sort by memory usage
	sort.Slice(allProcesses, func(i, j int) bool {
		return allProcesses[i].Process.Memory > allProcesses[j].Process.Memory
	})

	// Limit to top 10
	if len(allProcesses) > 10 {
		allProcesses = allProcesses[:10]
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if len(allProcesses) == 0 {
		fmt.Fprint(w, `
		<table class="table table-vcenter card-table">
			<thead>
				<tr>
					<th>Asset</th>
					<th>Process Name</th>
					<th>PID</th>
					<th>CPU %</th>
					<th>Memory</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td colspan="5" class="text-center text-muted py-4">
						No process data available
					</td>
				</tr>
			</tbody>
		</table>`)
		return
	}

	fmt.Fprint(w, `
	<table class="table table-vcenter card-table">
		<thead>
			<tr>
				<th>Asset</th>
				<th>Process Name</th>
				<th>PID</th>
				<th>CPU %</th>
				<th>Memory</th>
			</tr>
		</thead>
		<tbody>`)

	for _, p := range allProcesses {
		memStr := formatBytes(p.Process.Memory)
		fmt.Fprintf(w, `
		<tr>
			<td>%s</td>
			<td><code>%s</code></td>
			<td class="text-secondary">%d</td>
			<td>%.1f%%</td>
			<td>%s</td>
		</tr>`, p.Hostname, p.Process.Name, p.Process.PID, p.Process.CPU, memStr)
	}

	fmt.Fprint(w, `
		</tbody>
	</table>`)
}

// renderAlertsTable renders the HTMX partial for alerts table
func renderAlertsTable(w http.ResponseWriter) {
	alerts, err := store.GetAlerts(10, false)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if len(alerts) == 0 {
		fmt.Fprint(w, `
		<table class="table table-vcenter card-table">
			<thead>
				<tr>
					<th>Severity</th>
					<th>Asset</th>
					<th>Alert</th>
					<th>Time</th>
					<th>Action</th>
				</tr>
			</thead>
			<tbody>
				<tr>
					<td colspan="5" class="text-center text-muted py-4">
						<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-check me-2" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
							<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
							<path d="M5 12l5 5l10 -10"/>
						</svg>
						No active alerts - all systems normal
					</td>
				</tr>
			</tbody>
		</table>`)
		return
	}

	fmt.Fprint(w, `
	<table class="table table-vcenter card-table">
		<thead>
			<tr>
				<th>Severity</th>
				<th>Asset</th>
				<th>Alert</th>
				<th>Time</th>
				<th>Action</th>
			</tr>
		</thead>
		<tbody>`)

	for _, alert := range alerts {
		severityClass := "bg-warning"
		severityIcon := "alert-triangle"
		if alert.Severity == "critical" {
			severityClass = "bg-danger"
			severityIcon = "alert-octagon"
		}

		fmt.Fprintf(w, `
		<tr>
			<td>
				<span class="badge %s">
					<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-%s" width="16" height="16" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none">
						<path stroke="none" d="M0 0h24v24H0z" fill="none"/>
						<path d="M12 9v2m0 4v.01"/>
						<path d="M5 19h14a2 2 0 0 0 1.84 -2.75l-7.1 -12.25a2 2 0 0 0 -3.5 0l-7.1 12.25a2 2 0 0 0 1.75 2.75"/>
					</svg>
					%s
				</span>
			</td>
			<td><strong>%s</strong></td>
			<td>%s</td>
			<td class="text-secondary">%s</td>
			<td>
				<button class="btn btn-sm btn-outline-success" 
					hx-post="/api/alerts/resolve" 
					hx-vals='{"id": %d}'
					hx-swap="none"
					hx-on::after-request="htmx.trigger('#alerts-container', 'refresh')">
					Resolve
				</button>
			</td>
		</tr>`,
			severityClass, severityIcon, alert.Severity,
			alert.Hostname, alert.Message,
			formatTimeAgo(alert.CreatedAt), alert.ID)
	}

	fmt.Fprint(w, `
		</tbody>
	</table>`)
}
