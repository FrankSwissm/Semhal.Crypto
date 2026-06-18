from flask import Flask, render_template, jsonify, request, session, redirect, url_for
from flask_sqlalchemy import SQLAlchemy
import os

app = Flask(__name__, template_folder='templates', static_folder='static')
app.secret_key = 'SEMHAL_SYSTEM_ENCRYPTION_KEY_SECRET'

# Session Configuration
app.config.update(
    SESSION_COOKIE_HTTPONLY=True,
    SESSION_COOKIE_SAMESITE='Lax',
    SESSION_COOKIE_SECURE=False,
    PERMANENT_SESSION_LIFETIME=3600
)

# --- DATABASE CONFIGURATION ---
app.config['SQLALCHEMY_DATABASE_URI'] = os.environ.get('DATABASE_URL')
app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
db = SQLAlchemy(app)

class Account(db.Model):
    __tablename__ = 'accounts'
    address = db.Column(db.String(42), primary_key=True)
    balance = db.Column(db.Float, default=0.0)

def get_or_create_account(address):
    acc = Account.query.get(address)
    if not acc:
        acc = Account(address=address, balance=0.0)
        db.session.add(acc)
        db.session.commit()
    return acc

# --- INITIALIZATION ---
with app.app_context():
    db.create_all()

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
    return redirect(url_for('home'))

# --- PORTALS ---
@app.route('/portal/user')
def user_portal():
    if 'node_address' not in session: return redirect(url_for('home'))
    acc = Account.query.get(session['node_address'])
    return render_template('user_portal.html', address=session['node_address'], balance=acc.balance if acc else 0)

@app.route('/portal/miner')
def miner_portal():
    if 'node_address' not in session: return redirect(url_for('home'))
    acc = get_or_create_account(session['node_address'])
    return render_template('miner_portal.html', address=session['node_address'], balance=acc.balance)

# --- API LAYER ---
@app.route('/api/mine-reward', methods=['POST'])
def api_mine_reward():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    miner = get_or_create_account(session['node_address'])
    reward = 0.025
    miner.balance += reward
    db.session.commit()
    return jsonify({"status": "success", "reward": reward, "total": miner.balance})

@app.route('/api/transfer', methods=['POST'])
def api_transfer():
    if 'node_address' not in session: return jsonify({"status": "error", "message": "Unauthorized"}), 401
    sender = get_or_create_account(session['node_address'])
    recipient = get_or_create_account(request.form.get('recipient', '').strip())
    try:
        amount = float(request.form.get('amount', 0))
    except ValueError:
        return jsonify({"status": "error", "message": "Invalid amount"}), 400
    
    if session.get('role') != 'Admin':
        if amount < 0.0000001:
            return jsonify({"status": "error", "message": "Minimum send is 0.0000001"}), 400
        if sender.balance < amount:
            return jsonify({"status": "error", "message": "Insufficient balance"}), 400
        sender.balance -= amount
    
    recipient.balance += amount
    db.session.commit()
    return jsonify({"status": "success", "new_balance": sender.balance if session.get('role') != 'Admin' else 0})

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8085)
