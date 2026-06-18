from flask import Flask, render_template, jsonify, request, session, redirect, url_for
from flask_sqlalchemy import SQLAlchemy
import os

app = Flask(__name__, template_folder='templates', static_folder='static')
app.secret_key = 'SEMHAL_SYSTEM_ENCRYPTION_KEY_SECRET'

# Session Persistence Configuration
app.config.update(
    SESSION_COOKIE_HTTPONLY=True,
    SESSION_COOKIE_SAMESITE='Lax',
    SESSION_COOKIE_SECURE=False,
    PERMANENT_SESSION_LIFETIME=3600
)

# --- DATABASE INITIALIZATION ---
from flask_sqlalchemy import SQLAlchemy
import os

db = SQLAlchemy()

def init_app_database(app):
    """Call this function once when the app is initialized."""
    with app.app_context():
        db.create_all()
        defaults = {
            "0x1F98431c8aD98523631AE4a59f267346ea31F984": 500000000.0,
            "0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe": 1200000000.0,
            "0x71C7656EC7ab88b098defB751B7401B5f6d1476B": 350000000.0
        }
        for addr, bal in defaults.items():
            if not Account.query.get(addr):
                db.session.add(Account(address=addr, balance=bal))
        db.session.commit()

# --- Inside your main app setup ---
app = Flask(__name__)
app.config['SQLALCHEMY_DATABASE_URI'] = os.environ.get('DATABASE_URL')
db.init_app(app) # Initialize the extension safely

# DO NOT put the 'with app.app_context()' block here.
  

# --- GLOBAL MIDDLEWARE ---
@app.context_processor
def inject_auth_status():
    return dict(is_logged_in='node_address' in session, current_role=session.get('role'))

# --- NAVIGATION ROUTES ---
@app.route('/')
def home():
    total_nodes = Account.query.count()
    total_supply = db.session.query(db.func.sum(Account.balance)).scalar() or 0
    return render_template('index.html', total_nodes=total_nodes, total_supply=total_supply)

@app.route('/explorer')
def explorer():
    return render_template('explorer.html', ledger={a.address: a.balance for a in Account.query.all()})

@app.route('/docs')
def docs(): return render_template('docs.html')

@app.route('/ussd')
def ussd(): return render_template('ussd.html')

@app.route('/core')
def core(): return render_template('core.html')

@app.route('/markets')
def markets(): return render_template('markets.html')

@app.route('/news')
def news(): return render_template('news.html')

# --- AUTH & PORTALS ---
@app.route('/auth/login', methods=['POST'])
def auth_login():
    address = request.form.get('address', '').strip()
    password = request.form.get('password', '')
    role = "Admin" if password == "admin123" else ("Miner" if password == "miner123" else "User")
    session.permanent = True
    session['node_address'] = address
    session['role'] = role
    get_or_create_account(address)
    return jsonify({"status": "success", "role": role, "redirect": f"/portal/{role.lower()}"})

@app.route('/auth/logout')
def auth_logout():
    session.clear()
    return redirect(url_for('news'))

@app.route('/portal/user')
def user_portal():
    if 'node_address' not in session: return redirect(url_for('news'))
    acc = Account.query.get(session['node_address'])
    return render_template('user_portal.html', address=session['node_address'], balance=acc.balance if acc else 0)

@app.route('/api/mine-reward', methods=['POST'])
def api_mine_reward():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    miner = get_or_create_account(session['node_address'])
    
    # Fixed reward constant
    reward = 0.025
    
    miner.balance += reward
    db.session.commit()
    return jsonify({"status": "success", "reward": reward, "total": miner.balance})

@app.route('/portal/admin')
def admin_portal():
    if session.get('role') != 'Admin': return redirect(url_for('news'))
    return render_template('admin_portal.html', ledger={a.address: a.balance for a in Account.query.all()})

# --- API LAYER ---
@app.route('/api/transfer', methods=['POST'])
def api_transfer():
    if 'node_address' not in session: return jsonify({"status": "error", "message": "Unauthorized"}), 401
    
    sender = get_or_create_account(session['node_address'])
    recipient = get_or_create_account(request.form.get('recipient', '').strip())
    
    try:
        amount = float(request.form.get('amount', 0))
    except ValueError:
        return jsonify({"status": "error", "message": "Invalid amount"}), 400
    

    # Rule: Minimum send 0.0000001 sUSD (Admin bypass enabled)
    if session.get('role') != 'Admin':
        if amount < 0.0000001:
            return jsonify({"status": "error", "message": "Minimum send is 0.0000001 sUSD"}), 400
        if sender.balance < amount:
            return jsonify({"status": "error", "message": "Insufficient balance"}), 400
        sender.balance -= amount
    
    # Admin logic: Admins inject without sender balance checks
    recipient.balance += amount
    db.session.commit()
    return jsonify({"status": "success", "new_balance": sender.balance if session.get('role') != 'Admin' else 0})

@app.route('/api/mine-reward', methods=['POST'])
def api_mine_reward():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    miner = get_or_create_account(session['node_address'])
    miner.balance += 0.025
    db.session.commit()
    return jsonify({"status": "success", "reward": 0.025, "total": miner.balance})

@app.route('/api/admin/purge', methods=['POST'])
def api_admin_purge():
    if session.get('role') != 'Admin': return jsonify({"status": "error"}), 403
    target = Account.query.get(request.form.get('target', '').strip())
    if target:
        db.session.delete(target)
        db.session.commit()
        return jsonify({"status": "success"})
    return jsonify({"status": "error"}), 404

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8085)
