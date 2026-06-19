from flask import Flask, render_template, jsonify, request, session, redirect, url_for
from flask_sqlalchemy import SQLAlchemy
import os

app = Flask(__name__, template_folder='templates', static_folder='static')
app.secret_key = 'SEMHAL_SYSTEM_ENCRYPTION_KEY_SECRET'

# Database Configuration
app.config['SQLALCHEMY_DATABASE_URI'] = os.environ.get('DATABASE_URL')
app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
db = SQLAlchemy(app)

class Account(db.Model):
    __tablename__ = 'accounts'
    address = db.Column(db.String(42), primary_key=True)
    balance = db.Column(db.Float, default=0.0)

# --- INITIALIZATION ---
with app.app_context():
    # create_all() only creates tables if they don't exist.
    # It does NOT delete existing data, so it is safe to keep.
    db.create_all()
    
    # Optional: If you want to ensure default accounts exist,
    # wrap it in a check so you don't overwrite user balances.
    if not Account.query.first():
        # You can add your default accounts here if needed
        pass


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
    
    # Simple role assignment
    role = "Admin" if password == "admin123" else ("Miner" if password == "miner123" else "User")
    
    # Store in session
    session.permanent = True
    session['node_address'] = address
    session['role'] = role
    
    # Ensure account exists in DB
    get_or_create_account(address)
    
    # Return JSON for the frontend script to handle the redirect
    return jsonify({"status": "success", "redirect": f"/portal/{role.lower()}"})


@app.route('/portal/miner')
def miner_portal():
    if 'node_address' not in session: return redirect(url_for('news'))
    acc = get_or_create_account(session['node_address'])
    return render_template('miner_portal.html', address=session['node_address'], balance=acc.balance)

@app.route('/portal/admin')
def admin_portal():
    if session.get('role') != 'Admin': return redirect(url_for('news'))
    return render_template('admin_portal.html', ledger={a.address: a.balance for a in Account.query.all()})

# --- API LAYER ---
@app.route('/api/mine-reward', methods=['POST'])
def api_mine_reward():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    miner = Account.query.filter_by(address=session['node_address']).first()
    if not miner: miner = get_or_create_account(session['node_address'])
    
    new_balance = float(miner.balance) + 0.025
    db.session.query(Account).filter(Account.address == session['node_address']).update({"balance": new_balance})
    db.session.commit()
    return jsonify({"status": "success", "reward": 0.025, "total": new_balance})

@app.route('/api/transfer', methods=['POST'])
def api_transfer():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    sender = Account.query.filter_by(address=session['node_address']).first()
    recipient = get_or_create_account(request.form.get('recipient', '').strip())
    
    try:
        amount = float(request.form.get('amount', 0))
    except: return jsonify({"status": "error", "message": "Invalid amount"}), 400
    
    if session.get('role') != 'Admin':
        if amount < 0.0000001 or not sender or sender.balance < amount:
            return jsonify({"status": "error", "message": "Insufficient/Invalid"}), 400
        sender.balance -= amount
    
    recipient.balance += amount
    db.session.commit()
    return jsonify({"status": "success", "new_balance": sender.balance if sender else 0})

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8085)
