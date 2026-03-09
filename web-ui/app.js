const terminal = document.getElementById('terminal');
const agentList = document.getElementById('agent-list');
const statusDot = document.getElementById('status-dot');
const statusText = document.getElementById('status-text');

let socket;
const agents = new Map();
let currentEditType = null;
let currentEditIndex = null;

function connect() {
    socket = new WebSocket('ws://' + window.location.host + '/ws');

    socket.onopen = () => {
        statusDot.classList.add('active');
        statusText.innerText = 'CONNECTED';
        addLog('system', 'WebSocket connection established. Streaming live telemetry.');
    };

    socket.onclose = () => {
        statusDot.classList.remove('active');
        statusText.innerText = 'DISCONNECTED';
        addLog('system', 'Connection lost. Retrying in 5 seconds...');
        setTimeout(connect, 5000);
    };

    socket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        handleEvent(data);
    };
}

function handleEvent(event) {
    const { type, agent_role, payload, timestamp } = event;
    const time = new Date(timestamp).toLocaleTimeString();

    switch (type) {
        case 'agent_started':
            updateAgent(agent_role, 'Working', payload.input);
            addLog('system', `Agent [${agent_role}] initialized task execution.`, 'success');
            break;
        case 'agent_thinking':
            updateAgent(agent_role, 'Thinking');
            addLog('thinking', `Agent [${agent_role}] is processing iteration #${payload.iteration}...`);
            break;
        case 'tool_started':
            updateAgent(agent_role, 'Using Tool');
            addLog('tool', `Agent [${agent_role}] invoking tool: ${payload.tool}`);
            break;
        case 'tool_finished':
            addLog('tool', `Tool [${payload.tool}] returned result: ${payload.result ? payload.result.substring(0, 100) + '...' : 'OK'}`);
            break;
        case 'agent_finished':
            updateAgent(agent_role, 'Idle');
            addLog('system', `Agent [${agent_role}] successfully completed task.`, 'success');
            fetchEntities();
            break;
        case 'task_started':
            addLog('system', `New Task Started: ${payload.description.substring(0, 100)}...`);
            break;
        case 'task_finished':
            addLog('system', `Task Completed successfully.`, 'success');
            fetchEntities();
            break;
        case 'review_requested':
            updateAgent(agent_role, 'Awaiting Review');
            addLog('system', `Agent [${agent_role}] requested human review for tool: ${payload.tool_name}`, 'thinking');
            showReviewModal(payload.review_id, agent_role, payload.tool_name, payload.input);
            break;
        case 'system_metrics':
            document.getElementById('stat-ram').innerText = `${payload.memory_mb} MB`;
            document.getElementById('stat-go').innerText = payload.goroutines;
            document.getElementById('stat-uptime').innerText = `${Math.floor(payload.uptime_secs)}s`;
            break;
    }
}

function showReviewModal(reviewId, agentRole, toolName, input) {
    // Generate a modal dynamically for HITL
    const modal = document.createElement('div');
    modal.className = 'review-modal';
    modal.innerHTML = `
        <div class="review-content glass-card">
            <h3>Human Review Required</h3>
            <p><strong>Agent:</strong> ${agentRole}</p>
            <p><strong>Tool:</strong> ${toolName}</p>
            <div class="input-preview">${JSON.stringify(input, null, 2)}</div>
            <div class="review-actions">
                <button class="btn approve" onclick="submitReview('${reviewId}', true, this)">Approve</button>
                <button class="btn reject" onclick="submitReview('${reviewId}', false, this)">Reject</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function submitReview(reviewId, approved, buttonEl) {
    const modal = buttonEl.closest('.review-modal');

    fetch('/api/review', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            review_id: reviewId,
            approved: approved
        })
    }).then(res => {
        if (res.ok) {
            modal.remove();
            addLog('system', `Human review submitted: ${approved ? 'APPROVED' : 'REJECTED'}`, approved ? 'success' : 'system');
        } else {
            console.error('Failed to submit review');
        }
    }).catch(err => {
        console.error('Network error returning review:', err);
    });
}

function fetchEntities() {
    fetch('/api/list')
        .then(res => {
            if (!res.ok) {
                throw new Error(`HTTP error! status: ${res.status}`);
            }
            return res.json();
        })
        .then(data => {
            renderEntities(data);
        })
        .catch(err => {
            console.error('Error fetching entities:', err);
            addLog('system', `Sync Error: Failed to fetch dashboard entities: ${err.message}`, 'error');
        });
}

function updateProcessType() {
    const processType = document.getElementById('config-process-type').value;
    fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ process_type: processType })
    })
        .then(res => res.json())
        .then(data => {
            addLog('system', `Process Mode changed to: ${processType.toUpperCase()}`, 'success');
        })
        .catch(err => console.error('Error updating process type:', err));
}

function renderEntities(data) {
    // Current Process Type sync
    if (data.config && data.config.process_type) {
        const select = document.getElementById('config-process-type');
        if (select && select.value !== data.config.process_type) {
            select.value = data.config.process_type;
        }
    }

    agentList.innerHTML = '';

    // Render Agents
    const taskAgentSelect = document.getElementById('task-agent');
    if (taskAgentSelect) {
        taskAgentSelect.innerHTML = '<option value="">Select an Agent...</option>';
    }

    if (data.agents && data.agents.length > 0) {
        data.agents.forEach((agent, index) => {
            if (agent && agent.role) {
                const isManager = agent.allow_delegation;
                const managerBadge = isManager ? '<span class="status-badge" style="background:rgba(99,102,241,0.2); color:#818cf8; border-color:rgba(99,102,241,0.3); font-size:0.6rem; padding:0.1rem 0.4rem; margin-left:0.5rem; vertical-align:middle;">MANAGER</span>' : '';
                addEntityItem('Agent', agent.role, 'Staged', index, false, managerBadge);

                // Populate task creation dropdown
                if (taskAgentSelect) {
                    const opt = document.createElement('option');
                    opt.value = agent.role;
                    opt.textContent = agent.role;
                    taskAgentSelect.appendChild(opt);
                }
            }
        });
    }

    // Render Tasks
    if (data.tasks && data.tasks.length > 0) {
        data.tasks.forEach((task, index) => {
            if (task && task.description) {
                const desc = task.description.length > 30 ? task.description.substring(0, 30) + '...' : task.description;
                let status = task.agent_role || 'No Agent';
                if (task.failed) status += ' [FAILED ❌]';
                else if (task.processed) status += ' [COMPLETED ✅]';
                addEntityItem('Task', desc, status, index, task.processed || task.failed);
            }
        });
    }

    // Render MCPs
    if (data.mcp && data.mcp.length > 0) {
        data.mcp.forEach((mcp, index) => {
            const cmd = mcp.command || mcp.server_url;
            addEntityItem('MCP', cmd, 'Connected', index);
        });
    }

    // Render A2A
    if (data.a2a && data.a2a.length > 0) {
        data.a2a.forEach((a2a, index) => {
            addEntityItem('A2A', `${a2a.sender} -> ${a2a.receiver}`, a2a.model, index);
        });
    }
}

function addEntityItem(type, name, subtext, index, isProcessed = false) {
    const el = document.createElement('div');
    el.className = 'agent-item';
    if (!name) name = 'Unnamed';
    if (type === 'Agent') {
        el.id = `agent-${name.replace(/\s+/g, '-')}`;
    }

    let extraButtons = '';
    if (type === 'Task' && isProcessed) {
        extraButtons = `<button class="btn" style="padding: 0.2rem 0.5rem; font-size: 0.6rem; background: #10b981;" onclick="viewTaskResult(${index})">View Result</button>`;
    }

    el.innerHTML = `
        <div class="agent-info">
            <h4 style="font-size: 0.8rem;"><span style="color: var(--accent-color); font-weight: 800; font-size: 0.65rem; margin-right: 0.5rem;">${type.toUpperCase()}</span> ${name}</h4>
            <p class="agent-status">${subtext}</p>
        </div>
        <div style="display: flex; gap: 0.5rem; align-items: center;">
            ${extraButtons}
            <button class="btn" style="padding: 0.2rem 0.5rem; font-size: 0.6rem; background: var(--accent-color);" onclick="editEntity('${type.toLowerCase()}', ${index})">Edit</button>
            <button class="delete-btn" onclick="deleteEntity('${type.toLowerCase()}', ${index})">Delete</button>
        </div>
    `;
    agentList.appendChild(el);
}

function viewTaskResult(index) {
    fetch('/api/list')
        .then(res => res.json())
        .then(data => {
            const task = data.tasks[index];
            if (!task) return;
            const contentBox = document.getElementById('task-result-content');
            if (task.failed && task.error) {
                contentBox.value = `❌ ERROR:\n${task.error}`;
            } else if (task.output) {
                if (typeof task.output === 'object') {
                    contentBox.value = JSON.stringify(task.output, null, 2);
                } else {
                    contentBox.value = task.output;
                }
            } else {
                contentBox.value = "No output recorded.";
            }
            document.getElementById('modal-view-result').classList.remove('hidden');
        });
}


function editEntity(type, index) {
    currentEditType = type;
    currentEditIndex = index;

    fetch('/api/list')
        .then(res => res.json())
        .then(data => {
            const list = type === 'agent' ? data.agents : (type === 'task' ? data.tasks : (type === 'mcp' ? data.mcp : data.a2a));
            const item = list[index];
            if (!item) return;

            if (type === 'agent') {
                document.getElementById('agent-role').value = item.role;
                document.getElementById('agent-goal').value = item.goal;
                document.getElementById('agent-backstory').value = item.backstory;
                document.getElementById('agent-provider').value = item.provider || 'openai';
                document.getElementById('agent-model').value = item.llm_model || '';
                document.getElementById('agent-maxrpm').value = item.max_rpm || 0;
                // Check tools
                const toolCheckboxes = document.querySelectorAll('input[name="tools"]');
                toolCheckboxes.forEach(cb => cb.checked = item.tools && item.tools.includes(cb.value));

                document.getElementById('modal-create-agent').classList.remove('hidden');
            } else if (type === 'task') {
                document.getElementById('task-desc').value = item.description;
                document.getElementById('task-agent').value = item.agent_role;
                document.getElementById('modal-create-task').classList.remove('hidden');
            }
            // Add more types as needed

            // Change button text
            const submitBtn = document.querySelector(`#modal-create-${type} .approve`);
            if (submitBtn) submitBtn.innerText = 'Save Changes';
        });
}

function resetForm(type) {
    const modal = document.getElementById(`modal-create-${type}`);
    const inputs = modal.querySelectorAll('input, textarea, select');
    inputs.forEach(input => {
        if (input.type === 'checkbox') input.checked = false;
        else if (input.tagName === 'SELECT') input.selectedIndex = 0;
        else input.value = '';
    });

    // Reset button text
    const submitBtn = modal.querySelector('.approve');
    if (submitBtn) submitBtn.innerText = type === 'mcp' ? 'Connect' : (type === 'a2a' ? 'Establish Bridge' : 'Create');

    currentEditType = null;
    currentEditIndex = null;
}

function deleteEntity(type, index) {
    if (!confirm(`Are you sure you want to delete this ${type}?`)) return;

    fetch('/api/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ type, index })
    }).then(res => {
        if (res.ok) {
            addLog('system', `Deleted ${type} at index ${index}`, 'system');
            fetchEntities();
        }
    });
}

function updateAgent(role, status, goal = '') {
    if (!role) return;
    // Keep for active execution updates if needed, but fetchEntities handles static list
    const agentEl = document.getElementById(`agent-${role.replace(/\s+/g, '-')}`);
    if (agentEl) {
        agentEl.querySelector('.agent-status').innerText = status;
    }
}

function addLog(type, message, category = '') {
    const entry = document.createElement('div');
    entry.className = `log-entry type-${category || type}`;

    const timeSpan = document.createElement('span');
    timeSpan.className = 'log-time';
    timeSpan.innerText = new Date().toLocaleTimeString();

    const typeSpan = document.createElement('span');
    typeSpan.className = 'log-type';
    typeSpan.innerText = `[${type.toUpperCase()}]`;

    entry.appendChild(timeSpan);
    entry.appendChild(typeSpan);
    entry.appendChild(document.createTextNode(message));

    terminal.appendChild(entry);
    terminal.scrollTop = terminal.scrollHeight;

    // Limit log entries
    if (terminal.childElementCount > 100) {
        terminal.removeChild(terminal.firstChild);
    }
}

function toggleRemoteMemoryField() {
    const memory = document.getElementById('agent-memory').value;
    const group = document.getElementById('remote-memory-group');
    if (memory === 'remote_(rest/grpc)') {
        group.classList.remove('hidden');
    } else {
        group.classList.add('hidden');
    }
}

function toggleExecution(start) {
    const endpoint = start ? '/api/start' : '/api/stop';
    fetch(endpoint, { method: 'POST' })
        .then(res => {
            if (!res.ok) {
                res.text().then(text => addLog('system', `Failed to ${start ? 'start' : 'stop'} execution: ${text}`, 'system'));
            } else {
                addLog('system', `Execution ${start ? 'STARTED' : 'STOPPED'} via Dashboard request.`, 'success');
            }
        })
        .catch(err => addLog('system', `Network request error: ${err}`, 'system'));
}

function fetchMetadata() {
    // Fetch available tools
    fetch('/api/tools')
        .then(res => res.json())
        .then(tools => {
            const list = document.getElementById('agent-tools-list');
            list.innerHTML = '';
            tools.forEach(tool => {
                const label = document.createElement('label');
                label.style.display = 'flex';
                label.style.display = 'flex';
                label.style.alignItems = 'center';
                label.style.gap = '0.5rem';
                label.style.fontSize = '0.75rem';
                label.innerHTML = `<input type="checkbox" name="tools" value="${tool}"> ${tool}`;
                list.appendChild(label);
            });
        });

    // Fetch memory providers
    fetch('/api/memory')
        .then(res => res.json())
        .then(providers => {
            const select = document.getElementById('agent-memory');
            // select.innerHTML = '<option value="none">None</option>'; // Keep reset if needed
            providers.forEach(p => {
                const opt = document.createElement('option');
                opt.value = p.toLowerCase().replace(/\s+/g, '_');
                opt.innerText = p;
                select.appendChild(opt);
            });
        });

    // Fetch LLM providers
    fetch('/api/providers')
        .then(res => res.json())
        .then(providers => {
            const select = document.getElementById('agent-provider');
            select.innerHTML = '';
            providers.forEach(p => {
                const opt = document.createElement('option');
                opt.value = p.toLowerCase().replace(/\s+/g, '_');
                opt.innerText = p;
                select.appendChild(opt);
            });
        });
}

function submitCreateAgent() {
    const role = document.getElementById('agent-role').value;
    const goal = document.getElementById('agent-goal').value;
    const backstory = document.getElementById('agent-backstory').value;
    const provider = document.getElementById('agent-provider').value;
    const model = document.getElementById('agent-model').value;
    const apikey = document.getElementById('agent-apikey').value;
    const maxrpm = parseInt(document.getElementById('agent-maxrpm').value) || 0;
    const memory = document.getElementById('agent-memory').value;
    const connStr = document.getElementById('agent-remote-conn').value;
    const allowDelegation = document.getElementById('agent-delegation').checked;
    const selfHealing = document.getElementById('agent-selfhealing').checked;

    const selectedTools = [];
    document.querySelectorAll('.tool-checkbox:checked').forEach(cb => {
        selectedTools.push(cb.value);
    });

    const payload = {
        role,
        goal,
        backstory,
        provider,
        llm_model: model,
        api_key: apikey,
        max_rpm: maxrpm,
        memory,
        memory_config: { connection_string: connStr },
        tools: selectedTools,
        allow_delegation: allowDelegation,
        self_healing: selfHealing
    };

    fetch('/api/create/agent', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            ...payload,
            index: currentEditIndex
        })
    }).then(res => {
        if (res.ok) {
            document.getElementById('modal-create-agent').classList.add('hidden');
            addLog('system', `${currentEditIndex !== null ? 'Updated' : 'Created/staged'} Agent: ${role}`, 'success');
            resetForm('agent');
            fetchEntities();
        }
    }).catch(err => console.error(err));
}

function submitCreateTask() {
    const description = document.getElementById('task-desc').value;
    const agentRole = document.getElementById('task-agent').value;
    const expectedOutput = document.getElementById('task-expected').value;
    const asyncExecution = document.getElementById('task-async').checked;
    const timeout = parseInt(document.getElementById('task-timeout').value) || 120;

    const payload = {
        description,
        agent_role: agentRole,
        expected_output: expectedOutput,
        async_execution: asyncExecution,
        timeout: timeout
    };

    fetch('/api/create/task', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            ...payload,
            index: currentEditIndex
        })
    }).then(res => {
        if (res.ok) {
            document.getElementById('modal-create-task').classList.add('hidden');
            addLog('system', `${currentEditIndex !== null ? 'Updated' : 'Created/staged'} Task for Agent [${agentRole || 'First Available'}]`, 'success');
            resetForm('task');
            fetchEntities();
        }
    }).catch(err => console.error(err));
}

function submitCreateMCP() {
    const name = document.getElementById('mcp-name').value;
    const command = document.getElementById('mcp-command').value;
    const args = document.getElementById('mcp-args').value.split(',').map(a => a.trim());

    fetch('/api/create/mcp', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            name, command, args,
            index: currentEditIndex
        })
    }).then(res => {
        if (res.ok) {
            document.getElementById('modal-create-mcp').classList.add('hidden');
            addLog('system', `${currentEditIndex !== null ? 'Updated' : 'Connected to'} MCP Server: ${name}`, 'success');
            resetForm('mcp');
            fetchEntities();
        }
    });
}

function submitCreateA2A() {
    const sender = document.getElementById('a2a-sender').value;
    const receiver = document.getElementById('a2a-receiver').value;
    const model = document.getElementById('a2a-model').value;

    fetch('/api/create/a2a', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            sender, receiver, model,
            index: currentEditIndex
        })
    }).then(res => {
        if (res.ok) {
            document.getElementById('modal-create-a2a').classList.add('hidden');
            addLog('system', `${currentEditIndex !== null ? 'Updated' : 'Established'} A2A Bridge from [${sender}] to [${receiver}]`, 'success');
            resetForm('a2a');
            fetchEntities();
        }
    });
}

connect();
fetchMetadata();
fetchEntities();
