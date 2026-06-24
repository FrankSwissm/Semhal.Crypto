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
    password_changed = db.Column(db.Boolean, default=False)
    is_org = db.Column(db.Boolean, default=False)

with app.app_context():
    db.create_all()

def get_or_create_account(address):
    acc = Account.query.get(address)
    if not acc:
        acc = Account(address=address, balance=0.0)
        db.session.add(acc)
        db.session.commit()
    return acc

# --- NAVIGATION ROUTES ---
@app.route('/')
def home():
    total_nodes = Account.query.count()
    total_supply = db.session.query(db.func.sum(Account.balance)).scalar() or 0
    return render_template('index.html', total_nodes=total_nodes, total_supply=total_supply)

@app.route('/explorer')
def explorer():
    return render_template('explorer.html', ledger={acc.address: acc.balance for acc in Account.query.all()})

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
    acc = get_or_create_account(address)
    
    if password == "Organization@portal":
        role = "Organization"
        acc.is_org = True
        db.session.commit()
    elif acc.is_org:
        role = "Organization"
    elif password == "admin123":
        role = "Admin"
    elif password == "miner123":
        role = "Miner"
    else:
        role = "User"
        
    session.permanent = True
    session['node_address'] = address
    session['role'] = role
    
    if role == "Organization" and not acc.password_changed:
        return jsonify({"status": "success", "redirect": "/auth/change-password"})
    return jsonify({"status": "success", "redirect": f"/portal/{role.lower()}"})

@app.route('/auth/change-password', methods=['GET', 'POST'])
def change_password_page():
    if 'node_address' not in session: return redirect(url_for('home'))
    if request.method == 'POST':
        acc = get_or_create_account(session['node_address'])
        acc.password_changed = True
        session['role'] = 'Organization'
        db.session.commit()
        return redirect(url_for('organization_portal'))
    return render_template('change_password.html')

@app.route('/portal/organization')
def organization_portal():
    if session.get('role') != 'Organization': return redirect(url_for('news'))
    acc = get_or_create_account(session['node_address'])
    if not acc.password_changed: return redirect(url_for('change_password_page'))
    return render_template('organization_portal.html', address=session['node_address'], balance=acc.balance)

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
@app.route('/api/ai-monitor', methods=['GET'])
def api_ai_monitor():
    malicious = Account.query.filter(Account.balance < 0).all()
    for acc in malicious:
        acc.balance = 0.0
    db.session.commit()
    return jsonify({"malicious_detected": len(malicious) > 0})

@app.route('/api/transfer', methods=['POST'])
def api_transfer():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    sender = get_or_create_account(session['node_address'])
    recipient = get_or_create_account(request.form.get('recipient', '').strip())
    try:
        amount = float(request.form.get('amount', 0))
    except: return jsonify({"status": "error"}), 400
    if session.get('role') != 'Admin' and (amount < 0 or sender.balance < amount):
        return jsonify({"status": "error"}), 400
    sender.balance -= amount
    recipient.balance += amount
    db.session.commit()
    return jsonify({"status": "success", "new_balance": sender.balance})

@app.route('/auth/logout')
def auth_logout():
    session.clear()
    return redirect(url_for('home'))

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8085)
