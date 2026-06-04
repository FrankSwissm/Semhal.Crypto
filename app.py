from flask import Flask, render_template, jsonify, request, session, redirect, url_for
import json
import os

app = Flask(__name__, template_folder='templates', static_folder='static')
app.secret_key = 'SEMHAL_SYSTEM_ENCRYPTION_KEY_SECRET'

LEDGER_PATH = os.path.join(os.path.dirname(__file__), 'ledger.json')

def load_ledger():
    if os.path.exists(LEDGER_PATH):
        try:
            with open(LEDGER_PATH, 'r') as f:
                return json.load(f)
        except Exception:
            pass
    # Initialize default systemic accounts
    default_ledger = {
        "0x1F98431c8aD98523631AE4a59f267346ea31F984": 500000000,
        "0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe": 1200000000,
        "0x71C7656EC7ab88b098defB751B7401B5f6d1476B": 350000000
    }
    save_ledger(default_ledger)
    return default_ledger

def save_ledger(data):
    with open(LEDGER_PATH, 'w') as f:
        json.dump(data, f, indent=4)

# Global Navigation Middleware
@app.context_processor
def inject_auth_status():
    return dict(is_logged_in='node_address' in session, current_role=session.get('role'))

@app.route('/')
def home():
    ledger = load_ledger()
    return render_template('index.html', total_nodes=len(ledger), total_supply=sum(ledger.values()))

@app.route('/explorer')
def explorer():
    ledger = load_ledger()
    return render_template('explorer.html', ledger=ledger)

@app.route('/docs')
def docs():
    return render_template('docs.html')

@app.route('/ussd')
def ussd():
    return render_template('ussd.html')

@app.route('/core')
def core():
    return render_template('core.html')

@app.route('/markets')
def markets():
    return render_template('markets.html')

@app.route('/news')
def news():
    return render_template('news.html')

# --- AUTHENTICATION STATE OPERATIONS ---

@app.route('/auth/login', methods=['POST'])
def auth_login():
    address = request.form.get('address', '').strip()
    password = request.form.get('password', '')
    
    if not address.startswith("0x") or len(address) != 42:
        return jsonify({"status": "error", "message": "Invalid EIP-55 Node Signature Address structure."}), 400
        
    # Rule-Based Role Assignment Strategy for development environment testing
    if password == "admin123":
        role = "Admin"
    elif password == "miner123":
        role = "Miner"
    else:
        role = "User"
        
    session['node_address'] = address
    session['role'] = role
    
    # Guarantee account exists inside ledger registry
    ledger = load_ledger()
    if address not in ledger:
        ledger[address] = 1000000 # Give new profiles a starting baseline allocation
        save_ledger(ledger)
        
    return jsonify({"status": "success", "role": role, "redirect": f"/portal/{role.lower()}"})

@app.route('/auth/logout')
def auth_logout():
    session.clear()
    return redirect(url_for('news'))

@app.route('/auth/recovery', methods=['POST'])
def auth_recovery():
    address = request.form.get('address', '').strip()
    mnemonic = request.form.get('mnemonic', '').strip()
    
    if not address or len(mnemonic.split()) < 12:
        return jsonify({"status": "error", "message": "Recovery rejected. Ensure 12-word mnemonic phrase is populated."}), 400
        
    return jsonify({"status": "success", "message": "Passphrase access keys reset. Default security token assigned."})

# --- WORKSPACE ENDPOINTS ---

@app.route('/portal/user')
def user_portal():
    if 'node_address' not in session: return redirect(url_for('news'))
    ledger = load_ledger()
    balance = ledger.get(session['node_address'], 0)
    return render_template('user_portal.html', address=session['node_address'], balance=balance)

@app.route('/portal/miner')
def miner_portal():
    if 'node_address' not in session: return redirect(url_for('news'))
    return render_template('miner_portal.html', address=session['node_address'])

@app.route('/portal/admin')
def admin_portal():
    if 'node_address' not in session or session.get('role') != 'Admin': 
        return redirect(url_for('news'))
    ledger = load_ledger()
    return render_template('admin_portal.html', ledger=ledger)

# --- TRANSACTION API LAYER ACTIONS ---

@app.route('/api/transfer', methods=['POST'])
def api_transfer():
    if 'node_address' not in session: return jsonify({"status": "error", "message": "Unauthorized"}), 401
    
    sender = session['node_address']
    recipient = request.form.get('recipient', '').strip()
    try:
        amount = int(request.form.get('amount', 0))
    except ValueError:
        return jsonify({"status": "error", "message": "Invalid calculation metric."}), 400
        
    if sender == recipient:
        return jsonify({"status": "error", "message": "Self-loop transfers are restricted."}), 400
        
    ledger = load_ledger()
    if ledger.get(sender, 0) < amount or amount <= 0:
        return jsonify({"status": "error", "message": "Insufficient ledger allocation."}), 400
        
    ledger[sender] -= amount
    ledger[recipient] = ledger.get(recipient, 0) + amount
    save_ledger(ledger)
    
    return jsonify({"status": "success", "new_balance": ledger[sender]})

@app.route('/api/mine-reward', methods=['POST'])
def api_mine_reward():
    if 'node_address' not in session: return jsonify({"status": "error", "message": "Unauthorized"}), 401
    miner = session['node_address']
    
    ledger = load_ledger()
    reward = 25000000 # Flat systemic mint generation block reward
    ledger[miner] = ledger.get(miner, 0) + reward
    save_ledger(ledger)
    
    return jsonify({"status": "success", "reward": reward, "total": ledger[miner]})

@app.route('/api/admin/purge', methods=['POST'])
def api_admin_purge():
    if session.get('role') != 'Admin': return jsonify({"status": "error", "message": "Forbidden"}), 403
    target = request.form.get('target', '').strip()
    
    ledger = load_ledger()
    if target in ledger:
        del ledger[target]
        save_ledger(ledger)
        return jsonify({"status": "success"})
    return jsonify({"status": "error", "message": "Target index not discovered."}), 404

@app.route('/api/balances', methods=['GET'])
def get_balances():
    ledger_data = load_ledger()
    return jsonify({
        "status": "SECRET",
        "system_anchor": "Fixed Point",
        "total_supply_susd": sum(ledger_data.values()),
        "total_nodes": len(ledger_data),
        "accounts": ledger_data
    })

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8085, debug=True)
