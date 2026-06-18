from flask import Flask, render_template, jsonify, request, session, redirect, url_for
from flask_sqlalchemy import SQLAlchemy
import os

app = Flask(__name__, template_folder='templates', static_folder='static')
app.secret_key = 'SEMHAL_SYSTEM_ENCRYPTION_KEY_SECRET'

# Database Configuration (Uses the DATABASE_URL environment variable from Render)
app.config['SQLALCHEMY_DATABASE_URI'] = os.environ.get('DATABASE_URL')
app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
db = SQLAlchemy(app)

# Database Model
class Account(db.Model):
    __tablename__ = 'accounts'
    address = db.Column(db.String(42), primary_key=True)
    balance = db.Column(db.Float, default=0.00)

# Initialize database and seed defaults
with app.app_context():
    db.create_all()
    defaults = {
        "0x1F98431c8aD98523631AE4a59f267346ea31F984": 500000000,
        "0xde0B295669a9FD93d5F28D9Ec85E40f4cb697BAe": 1200000000,
        "0x71C7656EC7ab88b098defB751B7401B5f6d1476B": 350000000
    }
    for addr, bal in defaults.items():
        if not Account.query.get(addr):
            db.session.add(Account(address=addr, balance=bal))
    db.session.commit()

# Helper for account lookups
def get_or_create_account(address):
    acc = Account.query.get(address)
    if not acc:
        acc = Account(address=address, balance=0)
        db.session.add(acc)
        db.session.commit()
    return acc

# Global Navigation Middleware
@app.context_processor
def inject_auth_status():
    return dict(is_logged_in='node_address' in session, current_role=session.get('role'))

@app.route('/')
def home():
    total_nodes = Account.query.count()
    total_supply = db.session.query(db.func.sum(Account.balance)).scalar() or 0
    return render_template('index.html', total_nodes=total_nodes, total_supply=total_supply)

@app.route('/explorer')
def explorer():
    accounts = Account.query.all()
    ledger = {a.address: a.balance for a in accounts}
    return render_template('explorer.html', ledger=ledger)

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

# --- AUTHENTICATION ---

@app.route('/auth/login', methods=['POST'])
def auth_login():
    address = request.form.get('address', '').strip()
    password = request.form.get('password', '')
    if not address.startswith("0x") or len(address) != 42:
        return jsonify({"status": "error", "message": "Invalid address structure."}), 400

    role = "Admin" if password == "admin123" else ("Miner" if password == "miner123" else "User")
    session['node_address'] = address
    session['role'] = role
    get_or_create_account(address)
    return jsonify({"status": "success", "role": role, "redirect": f"/portal/{role.lower()}"})

@app.route('/auth/logout')
def auth_logout():
    session.clear()
    return redirect(url_for('news'))

# --- WORKSPACE ---

@app.route('/portal/user')
def user_portal():
    if 'node_address' not in session: return redirect(url_for('news'))
    acc = Account.query.get(session['node_address'])
    return render_template('user_portal.html', address=session['node_address'], balance=acc.balance if acc else 0)

@app.route('/portal/miner')
def miner_portal():
    if 'node_address' not in session: return redirect(url_for('news'))
    return render_template('miner_portal.html', address=session['node_address'])

@app.route('/portal/admin')
def admin_portal():
    if session.get('role') != 'Admin': return redirect(url_for('news'))
    accounts = Account.query.all()
    ledger = {a.address: a.balance for a in accounts}
    return render_template('admin_portal.html', ledger=ledger)

# --- TRANSACTION API ---

@app.route('/api/transfer', methods=['POST'])
def api_transfer():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    sender = get_or_create_account(session['node_address'])
    recipient = get_or_create_account(request.form.get('recipient', '').strip())
    amount = int(request.form.get('amount', 0))

    if sender.address == recipient.address or sender.balance < amount or amount <= 0.5:
        return jsonify({"status": "error", "message": "Invalid transaction"}), 400

    sender.balance -= amount
    recipient.balance += amount
    db.session.commit()
    return jsonify({"status": "success", "new_balance": sender.balance})

@app.route('/api/mine-reward', methods=['POST'])
def api_mine_reward():
    miner = get_or_create_account(session['node_address'])
    reward = 0.25
    miner.balance += reward
    db.session.commit()
    return jsonify({"status": "success", "reward": reward, "total": miner.balance})

@app.route('/api/admin/purge', methods=['POST'])
def api_admin_purge():
    if session.get('role') != 'Admin': return jsonify({"status": "error"}), 403
    target = Account.query.get(request.form.get('target', '').strip())
    if target:
        db.session.delete(target)
        db.session.commit()
        return jsonify({"status": "success"})
    return jsonify({"status": "error"}), 404

@app.route('/api/balances', methods=['GET'])
def get_balances():
    accounts = Account.query.all()
    ledger = {a.address: a.balance for a in accounts}
    return jsonify({"accounts": ledger, "total_supply": sum(ledger.values())})

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8085, debug=True)