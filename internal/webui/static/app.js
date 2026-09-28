const targetInput = document.getElementById('target')
const portsInput = document.getElementById('ports')
const timeoutSelect = document.getElementById('timeout')
const scanButton = document.getElementById('scanButton')
const exportButton = document.getElementById('exportButton')
const scanStatus = document.getElementById('scanStatus')
const portsTable = document.getElementById('portsTable')
const changes = document.getElementById('changes')
const history = document.getElementById('history')
const openCount = document.getElementById('openCount')
const newCount = document.getElementById('newCount')
const closedCount = document.getElementById('closedCount')
const duration = document.getElementById('duration')

let currentResult = null

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;')
}

function setBusy(busy) {
  scanButton.disabled = busy
  scanButton.textContent = busy ? 'Scanning…' : 'Scan target'
}

function renderPorts(result) {
  if (!result.openPorts || result.openPorts.length === 0) {
    portsTable.className = 'table empty'
    portsTable.textContent = 'No open TCP ports were found in the requested set.'
    return
  }

  portsTable.className = 'table'

  const rows = result.openPorts.map(item => `
    <div class="portRow">
      <div class="portNumber">${item.port}</div>
      <div class="serviceName">${escapeHtml(item.service)}</div>
      <div class="openState">OPEN</div>
    </div>
  `).join('')

  portsTable.innerHTML = `
    <div class="tableHeader">
      <div>PORT</div>
      <div>SERVICE</div>
      <div style="text-align:right">STATUS</div>
    </div>
    ${rows}
  `
}

function renderChanges(result) {
  if (!result.hasPrevious) {
    changes.innerHTML = '<div class="emptyBlock">Baseline created. Run the same target again to detect changes.</div>'
    return
  }

  if (result.newPorts.length === 0 && result.closedPorts.length === 0) {
    changes.innerHTML = '<div class="emptyBlock">No port changes since the previous scan.</div>'
    return
  }

  const newTags = result.newPorts
    .map(port => `<span class="tag new">+ ${port}</span>`)
    .join('')

  const closedTags = result.closedPorts
    .map(port => `<span class="tag closed">− ${port}</span>`)
    .join('')

  changes.innerHTML = `
    ${result.newPorts.length ? `
      <div class="changeGroup">
        <div class="changeTitle">NEWLY OPEN</div>
        <div class="tags">${newTags}</div>
      </div>
    ` : ''}
    ${result.closedPorts.length ? `
      <div class="changeGroup">
        <div class="changeTitle">CLOSED</div>
        <div class="tags">${closedTags}</div>
      </div>
    ` : ''}
  `
}

function renderResult(result) {
  currentResult = result
  exportButton.disabled = false

  openCount.textContent = result.openPorts.length
  newCount.textContent = result.hasPrevious ? result.newPorts.length : '—'
  closedCount.textContent = result.hasPrevious ? result.closedPorts.length : '—'
  duration.textContent = `${result.durationMs} ms`

  scanStatus.classList.remove('error')
  scanStatus.textContent = `${result.target} → ${result.resolvedIp} · ${result.requestedPorts} ports`

  renderPorts(result)
  renderChanges(result)
}

async function loadHistory() {
  try {
    const response = await fetch('/api/history?limit=12')
    if (!response.ok) {
      return
    }

    const items = await response.json()

    if (!Array.isArray(items) || items.length === 0) {
      history.innerHTML = '<div class="emptyBlock">No scans yet.</div>'
      return
    }

    history.innerHTML = items.map(item => `
      <div class="historyRow">
        <div>
          <strong>${escapeHtml(item.target)}</strong>
          <span>${new Date(item.startedAt).toLocaleString()}</span>
        </div>
        <div class="historyCount">${item.openCount} open · ${item.durationMs} ms</div>
      </div>
    `).join('')
  } catch {
    history.innerHTML = '<div class="emptyBlock">History unavailable.</div>'
  }
}

async function runScan() {
  const target = targetInput.value.trim()
  const ports = portsInput.value.trim()
  const timeoutMs = Number(timeoutSelect.value)

  if (!target || !ports) {
    scanStatus.classList.add('error')
    scanStatus.textContent = 'Target and ports are required.'
    return
  }

  setBusy(true)
  scanStatus.classList.remove('error')
  scanStatus.textContent = 'Scanning private target…'

  try {
    const response = await fetch('/api/scan', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ target, ports, timeoutMs })
    })

    const body = await response.json()

    if (!response.ok) {
      throw new Error(body.error || `HTTP ${response.status}`)
    }

    renderResult(body)
    await loadHistory()
  } catch (error) {
    scanStatus.classList.add('error')
    scanStatus.textContent = error instanceof Error ? error.message : 'Scan failed.'
  } finally {
    setBusy(false)
  }
}

function exportResult() {
  if (!currentResult) {
    return
  }

  const content = JSON.stringify(currentResult, null, 2)
  const blob = new Blob([content], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `socketlens-${currentResult.target}-${currentResult.id}.json`
  link.click()
  URL.revokeObjectURL(url)
}

scanButton.addEventListener('click', runScan)
exportButton.addEventListener('click', exportResult)

targetInput.addEventListener('keydown', event => {
  if (event.key === 'Enter') {
    runScan()
  }
})

loadHistory()
