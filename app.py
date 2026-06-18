from flask import Flask, render_template, jsonify, request, session, redirect, url_for
from flask_sqlalchemy import SQLAlchemy
import os

app = Flask(__name__, template_folder='templates', static_folder='static')
app.secret_key = 'SEMHAL_SYSTEM_ENCRYPTION_KEY_SECRET'

app.config['SQLALCHEMY_DATABASE_URI'] = os.environ.get('DATABASE_URL')
app.config['SQLALCHEMY_TRACK_MODIFICATIONS'] = False
db = SQLAlchemy(app)

class Account(db.Model):
    __tablename__ = 'accounts'
    address = db.Column(db.String(42), primary_key=True)
    # Changed default to 0.0 to meet requirement
    balance = db.Column(db.Float, default=0.0)

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

def get_or_create_account(address):
    acc = Account.query.get(address)
    if not acc:
        acc = Account(address=address, balance=0.0)
        db.session.add(acc)
        db.session.commit()
    return acc

# ... (Keep existing routes for /, /explorer, /docs, /ussd, /core, /markets, /news) ...

@app.route('/api/transfer', methods=['POST'])
def api_transfer():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    
    sender = get_or_create_account(session['node_address'])
    recipient = get_or_create_account(request.form.get('recipient', '').strip())
    
    try:
        amount = float(request.form.get('amount', 0))
    except ValueError:
        return jsonify({"status": "error", "message": "Invalid amount"}), 400

    # Rule: Minimum send 0.05 sUSD (Admin can bypass)
    if session.get('role') != 'Admin' and amount < 0.05:
        return jsonify({"status": "error", "message": "Minimum send requirement is 0.05 sUSD"}), 400

    if sender.address == recipient.address or sender.balance < amount or amount <= 0:
        return jsonify({"status": "error", "message": "Invalid transaction"}), 400

    sender.balance -= amount
    recipient.balance += amount
    db.session.commit()
    return jsonify({"status": "success", "new_balance": sender.balance})

@app.route('/api/mine-reward', methods=['POST'])
def api_mine_reward():
    if 'node_address' not in session: return jsonify({"status": "error"}), 401
    miner = get_or_create_account(session['node_address'])
    # Updated reward to 0.025
    reward = 0.025
    miner.balance += reward
    db.session.commit()
    return jsonify({"status": "success", "reward": reward, "total": miner.balance})

# ... (Keep remaining API routes) ...
