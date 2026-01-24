// 用户注册功能
async function registerUser() {
    const username = document.getElementById('username').value.trim();
    const resultDiv = document.getElementById('register-result');
    
    if (!username) {
        showResult(resultDiv, '请输入用户名', 'error');
        return;
    }
    
    try {
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username: username })
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            showResult(resultDiv, `注册成功！用户ID: ${data.user_id}`, 'success');
            document.getElementById('username').value = '';
            loadAllUsers(); // 刷新用户列表
        } else {
            showResult(resultDiv, `注册失败: ${data.message}`, 'error');
        }
    } catch (error) {
        showResult(resultDiv, `请求失败: ${error.message}`, 'error');
    }
}

// 用户签到功能
async function userCheckIn() {
    const userId = document.getElementById('checkin-user-id').value.trim();
    const resultDiv = document.getElementById('checkin-result');
    
    if (!userId) {
        showResult(resultDiv, '请输入用户ID', 'error');
        return;
    }
    
    try {
        const response = await fetch('/api/checkin', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ user_id: parseInt(userId) })
        });
        
        const data = await response.json();
        
        if (data.status === 'success') {
            showResult(resultDiv, data.message, 'success');
            document.getElementById('checkin-user-id').value = '';
            loadAllUsers(); // 刷新用户列表以显示更新后的积分
        } else {
            showResult(resultDiv, `签到失败: ${data.message}`, 'error');
        }
    } catch (error) {
        showResult(resultDiv, `请求失败: ${error.message}`, 'error');
    }
}

// 用户查询功能
async function queryUser() {
    const username = document.getElementById('query-username').value.trim();
    const resultDiv = document.getElementById('query-result');
    
    if (!username) {
        showResult(resultDiv, '请输入要查询的用户名', 'error');
        return;
    }
    
    try {
        const response = await fetch(`/api/user?username=${encodeURIComponent(username)}`);
        const data = await response.json();
        
        if (data.status === 'success') {
            const user = data.user;
            const userInfo = `
                <div class="user-info">
                    <h3>用户信息</h3>
                    <p><strong>ID:</strong> ${user.id}</p>
                    <p><strong>用户名:</strong> ${user.username}</p>
                    <p><strong>积分余额:</strong> ${user.balance || 0}</p>
                    <p><strong>注册时间:</strong> ${formatDate(user.created_at)}</p>
                    <p><strong>更新时间:</strong> ${formatDate(user.updated_at)}</p>
                </div>
            `;
            resultDiv.innerHTML = userInfo;
        } else {
            showResult(resultDiv, `查询失败: ${data.message}`, 'error');
        }
    } catch (error) {
        showResult(resultDiv, `请求失败: ${error.message}`, 'error');
    }
}

// 加载所有用户列表
async function loadAllUsers() {
    const usersListDiv = document.getElementById('users-list');
    
    try {
        const response = await fetch('/api/users');
        const data = await response.json();
        
        if (data.status === 'success') {
            const users = data.users;
            if (users.length === 0) {
                usersListDiv.innerHTML = '<p>暂无用户数据</p>';
                return;
            }
            
            let html = '<div class="users-table">';
            html += '<h3>用户列表</h3>';
            html += '<table><thead><tr><th>ID</th><th>用户名</th><th>积分余额</th><th>注册时间</th><th>操作</th></tr></thead><tbody>';
            
            users.forEach(user => {
                html += `
                    <tr>
                        <td>${user.id}</td>
                        <td>${user.username}</td>
                        <td>${user.balance || 0}</td>
                        <td>${formatDate(user.created_at)}</td>
                        <td>
                            <button onclick="quickCheckIn(${user.id})" class="btn-small">签到</button>
                            <button onclick="quickQuery('${user.username}')" class="btn-small">查询</button>
                        </td>
                    </tr>
                `;
            });
            
            html += '</tbody></table></div>';
            usersListDiv.innerHTML = html;
        } else {
            usersListDiv.innerHTML = `<p>加载失败: ${data.message}</p>`;
        }
    } catch (error) {
        usersListDiv.innerHTML = `<p>加载失败: ${error.message}</p>`;
    }
}

// 快速签到
function quickCheckIn(userId) {
    document.getElementById('checkin-user-id').value = userId;
    userCheckIn();
}

// 快速查询
function quickQuery(username) {
    document.getElementById('query-username').value = username;
    queryUser();
}

// 显示结果信息
function showResult(element, message, type) {
    element.innerHTML = `<div class="result-${type}">${message}</div>`;
    // 3秒后自动清除成功消息
    if (type === 'success') {
        setTimeout(() => {
            element.innerHTML = '';
        }, 3000);
    }
}

// 格式化日期
function formatDate(dateString) {
    if (!dateString) return '未知';
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN');
}

// 页面加载时自动加载用户列表
document.addEventListener('DOMContentLoaded', function() {
    loadAllUsers();
    
    // 为输入框添加回车键支持
    document.getElementById('username').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') registerUser();
    });
    
    document.getElementById('checkin-user-id').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') userCheckIn();
    });
    
    document.getElementById('query-username').addEventListener('keypress', function(e) {
        if (e.key === 'Enter') queryUser();
    });
});
