import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'i18n.dart';

import 'api.dart';
import 'controller.dart';
import 'links.dart';

/// The signed-in user's account: profile, password, two-factor sign-in and
/// recovery codes. Each section appears when the server offers it
/// (Panel.Accounts).
class AccountScreen extends StatefulWidget {
  const AccountScreen({super.key, required this.admin});

  final AdminController admin;

  @override
  State<AccountScreen> createState() => _AccountScreenState();
}

class _AccountScreenState extends State<AccountScreen> {
  Map<String, dynamic>? _account;
  String? _loadError;

  final _name = TextEditingController();
  final _email = TextEditingController();
  final _current = TextEditingController();
  final _password = TextEditingController();
  final _confirm = TextEditingController();
  bool _busy = false;

  /// Two-factor setup in progress: {secret, otpauth_uri}.
  Map<String, dynamic>? _setup;
  final _code = TextEditingController();

  /// Recovery codes just made, shown once.
  List<String>? _freshCodes;

  AdminApi get api => widget.admin.api;
  Map<String, dynamic> get _features {
    final f = _account?['features'];
    return f is Map ? Map<String, dynamic>.from(f) : const {};
  }

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    for (final c in [_name, _email, _current, _password, _confirm, _code]) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final a = await api.account();
      if (!mounted) return;
      setState(() {
        _account = a;
        _name.text = '${a['name'] ?? ''}';
        _email.text = '${a['email'] ?? ''}';
      });
    } on ApiException catch (e) {
      if (mounted) setState(() => _loadError = e.message);
    } catch (_) {
      if (mounted) {
        setState(() => _loadError = tr('Could not reach the server.'));
      }
    }
  }

  void _toast(String msg, {bool error = false}) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: error ? Theme.of(context).colorScheme.error : null,
    ));
  }

  Future<void> _run(Future<void> Function() fn) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await fn();
    } on ApiException catch (e) {
      if (mounted) _toast(e.message, error: true);
    } catch (_) {
      if (mounted) _toast(tr('Could not reach the server.'), error: true);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Asks for the current password before a sensitive change.
  Future<String?> _askPassword(String title) async {
    if (_features['password'] != true) return '';
    return showDialog<String>(
        context: context, builder: (_) => _PasswordPrompt(title: title));
  }

  void _takeCodes(Map<String, dynamic> reply) {
    final codes = reply['recovery_codes'];
    _account = {...?_account, ...reply}..remove('recovery_codes');
    _freshCodes = codes is List ? [for (final c in codes) '$c'] : null;
  }

  Future<void> _saveProfile() => _run(() async {
        final a =
            await api.updateAccount(_name.text.trim(), _email.text.trim());
        setState(() => _account = a);
        widget.admin.api.user = {
          ...?widget.admin.api.user,
          'name': a['name'],
          'email': a['email'],
        };
        _toast(tr('Your profile was saved.'));
      });

  Future<void> _changePassword() => _run(() async {
        await api.changePassword(_current.text, _password.text, _confirm.text);
        _current.clear();
        _password.clear();
        _confirm.clear();
        _toast(tr('Your password was changed.'));
      });

  Future<void> _startTwoFactor() async {
    final pw = await _askPassword(tr('Set up two-factor sign-in'));
    if (pw == null) return;
    await _run(() async {
      final setup = await api.twoFactorEnable(pw);
      setState(() {
        _setup = setup;
        _code.clear();
      });
    });
  }

  Future<void> _confirmTwoFactor() => _run(() async {
        final reply = await api.twoFactorConfirm(_code.text.trim());
        setState(() {
          _setup = null;
          _takeCodes(reply);
        });
        _toast(tr('Two-factor sign-in is on.'));
      });

  Future<void> _disableTwoFactor() async {
    final pw = await _askPassword(tr('Turn off two-factor sign-in'));
    if (pw == null) return;
    await _run(() async {
      final reply = await api.twoFactorDisable(pw);
      setState(() {
        _account = {...?_account, ...reply};
        _freshCodes = null;
      });
      _toast(tr('Two-factor sign-in is off.'));
    });
  }

  Future<void> _newCodes() async {
    final pw = await _askPassword(tr('Make new recovery codes'));
    if (pw == null) return;
    await _run(() async {
      final reply = await api.newRecoveryCodes(pw);
      setState(() => _takeCodes(reply));
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    Widget section(String title, List<Widget> children, {Key? key}) => Card(
          key: key,
          margin: const EdgeInsets.only(bottom: 16),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(title,
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 12),
                  ...children,
                ]),
          ),
        );
    const gap = SizedBox(height: 12);

    Widget body;
    if (_loadError != null) {
      body = Center(child: Text(_loadError!));
    } else if (_account == null) {
      body = const Center(child: CircularProgressIndicator());
    } else {
      final twoFactorOn = _account!['two_factor_enabled'] == true;
      body = ListView(padding: const EdgeInsets.all(16), children: [
        Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (_features['profile'] == true)
                    section('Profile', key: const Key('account-profile'), [
                      TextField(
                        key: const Key('account-name'),
                        controller: _name,
                        readOnly: _features['update_profile'] != true,
                        decoration: InputDecoration(labelText: tr('Name')),
                      ),
                      gap,
                      TextField(
                        key: const Key('account-email'),
                        controller: _email,
                        readOnly: _features['update_profile'] != true,
                        keyboardType: TextInputType.emailAddress,
                        decoration:
                            InputDecoration(labelText: tr('Email address')),
                      ),
                      if (_features['update_profile'] == true) ...[
                        gap,
                        Align(
                          alignment: Alignment.centerLeft,
                          child: FilledButton(
                              key: const Key('account-save'),
                              onPressed: _busy ? null : _saveProfile,
                              child: Text(tr('Save profile'))),
                        ),
                      ],
                    ]),
                  if (_features['password'] == true)
                    section(
                        tr('Change password'),
                        key: const Key('account-password'),
                        [
                          TextField(
                            key: const Key('account-current-password'),
                            controller: _current,
                            obscureText: true,
                            decoration: InputDecoration(
                                labelText: tr('Current password')),
                          ),
                          gap,
                          TextField(
                            key: const Key('account-new-password'),
                            controller: _password,
                            obscureText: true,
                            decoration:
                                InputDecoration(labelText: tr('New password')),
                          ),
                          gap,
                          TextField(
                            key: const Key('account-confirm-password'),
                            controller: _confirm,
                            obscureText: true,
                            decoration: InputDecoration(
                                labelText: tr('Confirm new password')),
                          ),
                          gap,
                          Align(
                            alignment: Alignment.centerLeft,
                            child: FilledButton(
                                key: const Key('account-change-password'),
                                onPressed: _busy ? null : _changePassword,
                                child: Text(tr('Change password'))),
                          ),
                        ]),
                  if (_features['two_factor'] == true)
                    section(
                        tr('Two-factor sign-in'),
                        key: const Key('account-two-factor'),
                        [
                          if (_setup != null) ...[
                            Text(tr(
                                'Add this key to your authenticator app, then enter the code it shows.')),
                            gap,
                            SelectableText('${_setup!['secret']}',
                                key: const Key('two-factor-secret'),
                                style: const TextStyle(
                                    fontFamily: 'monospace',
                                    fontSize: 16,
                                    letterSpacing: 1.5)),
                            Wrap(spacing: 8, children: [
                              TextButton.icon(
                                icon: const Icon(Icons.copy, size: 16),
                                label: Text(tr('Copy key')),
                                onPressed: () => Clipboard.setData(
                                    ClipboardData(
                                        text: '${_setup!['secret']}')),
                              ),
                              TextButton.icon(
                                icon: const Icon(Icons.open_in_new, size: 16),
                                label: Text(tr('Open in authenticator app')),
                                onPressed: () =>
                                    openLink('${_setup!['otpauth_uri']}'),
                              ),
                            ]),
                            TextField(
                              key: const Key('two-factor-code'),
                              controller: _code,
                              keyboardType: TextInputType.number,
                              decoration:
                                  InputDecoration(labelText: tr('Code')),
                              onSubmitted: (_) => _confirmTwoFactor(),
                            ),
                            gap,
                            Row(children: [
                              FilledButton(
                                  key: const Key('two-factor-confirm'),
                                  onPressed: _busy ? null : _confirmTwoFactor,
                                  child: Text(tr('Turn on'))),
                              const SizedBox(width: 8),
                              TextButton(
                                  onPressed: () =>
                                      setState(() => _setup = null),
                                  child: Text(tr('Cancel'))),
                            ]),
                          ] else if (twoFactorOn) ...[
                            Text(tr('Two-factor sign-in is on.')),
                            if (_freshCodes != null) ...[
                              gap,
                              Container(
                                key: const Key('recovery-codes'),
                                padding: const EdgeInsets.all(12),
                                decoration: BoxDecoration(
                                  color: theme.colorScheme.secondaryContainer,
                                  borderRadius: BorderRadius.circular(10),
                                ),
                                child: Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.start,
                                    children: [
                                      Text(tr('Save these recovery codes. Each one signs you in once if you lose your phone. They will not be shown again.'),
                                          style: TextStyle(
                                              fontWeight: FontWeight.w600)),
                                      const SizedBox(height: 8),
                                      SelectableText(_freshCodes!.join('\n'),
                                          style: const TextStyle(
                                              fontFamily: 'monospace')),
                                      TextButton.icon(
                                        icon: const Icon(Icons.copy, size: 16),
                                        label: Text(tr('Copy codes')),
                                        onPressed: () => Clipboard.setData(
                                            ClipboardData(
                                                text: _freshCodes!.join('\n'))),
                                      ),
                                    ]),
                              ),
                            ],
                            if (_features['recovery_codes'] == true) ...[
                              gap,
                              Text(
                                  tr('Recovery codes left: {n}', {
                                    'n': _account!['recovery_codes_left'] ?? 0
                                  }),
                                  key: const Key('recovery-codes-left')),
                            ],
                            gap,
                            Wrap(spacing: 8, runSpacing: 8, children: [
                              if (_features['recovery_codes'] == true)
                                OutlinedButton(
                                    key: const Key('recovery-codes-new'),
                                    onPressed: _busy ? null : _newCodes,
                                    child: Text(tr('Make new recovery codes'))),
                              OutlinedButton(
                                  key: const Key('two-factor-disable'),
                                  style: OutlinedButton.styleFrom(
                                      foregroundColor: theme.colorScheme.error),
                                  onPressed: _busy ? null : _disableTwoFactor,
                                  child: Text(tr('Turn off'))),
                            ]),
                          ] else ...[
                            Text(tr(
                                'Ask for a code from an authenticator app when you sign in.')),
                            gap,
                            Align(
                              alignment: Alignment.centerLeft,
                              child: FilledButton(
                                  key: const Key('two-factor-setup'),
                                  onPressed: _busy ? null : _startTwoFactor,
                                  child: Text(tr('Set up'))),
                            ),
                          ],
                        ]),
                ]),
          ),
        ),
      ]);
    }
    return Scaffold(
      appBar: AppBar(title: Text(tr('Your account'))),
      body: body,
    );
  }
}

class _PasswordPrompt extends StatefulWidget {
  const _PasswordPrompt({required this.title});
  final String title;

  @override
  State<_PasswordPrompt> createState() => _PasswordPromptState();
}

class _PasswordPromptState extends State<_PasswordPrompt> {
  final _c = TextEditingController();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: Text(widget.title),
        content: TextField(
          key: const Key('confirm-password'),
          controller: _c,
          obscureText: true,
          autofocus: true,
          decoration: InputDecoration(labelText: tr('Current password')),
          onSubmitted: (v) => Navigator.pop(context, v),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: Text(tr('Cancel'))),
          FilledButton(
              key: const Key('confirm-password-ok'),
              onPressed: () => Navigator.pop(context, _c.text),
              child: Text(tr('Continue'))),
        ],
      );
}

/// Shown after sign-in while the account's email address is not verified.
class VerifyEmailScreen extends StatefulWidget {
  const VerifyEmailScreen({super.key, required this.admin});
  final AdminController admin;

  @override
  State<VerifyEmailScreen> createState() => _VerifyEmailScreenState();
}

class _VerifyEmailScreenState extends State<VerifyEmailScreen> {
  bool _busy = false;
  String? _message;

  Future<void> _resend() async {
    setState(() => _busy = true);
    try {
      final verified = await widget.admin.api.resendVerification();
      if (verified) {
        await widget.admin.refresh();
        return;
      }
      _message = tr('A new verification link was sent to your email address.');
    } on ApiException catch (e) {
      _message = e.message;
    } catch (_) {
      _message = tr('Could not reach the server.');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Card(
            margin: const EdgeInsets.all(20),
            child: Padding(
              padding: const EdgeInsets.all(28),
              child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(tr('Verify your email'),
                        style: theme.textTheme.titleLarge
                            ?.copyWith(fontWeight: FontWeight.w700)),
                    const SizedBox(height: 12),
                    Text(tr(
                        'Before you continue, open the link we emailed you to verify your address.')),
                    if (_message != null) ...[
                      const SizedBox(height: 12),
                      Text(_message!,
                          key: const Key('verify-message'),
                          style: TextStyle(color: theme.colorScheme.primary)),
                    ],
                    const SizedBox(height: 20),
                    FilledButton(
                      key: const Key('verify-continue'),
                      onPressed: _busy ? null : widget.admin.refresh,
                      child: Text(tr('I have verified it')),
                    ),
                    const SizedBox(height: 8),
                    OutlinedButton(
                      key: const Key('verify-resend'),
                      onPressed: _busy ? null : _resend,
                      child: Text(tr('Send the link again')),
                    ),
                    TextButton(
                      key: const Key('verify-logout'),
                      onPressed: widget.admin.logout,
                      child: Text(tr('Sign out')),
                    ),
                  ]),
            ),
          ),
        ),
      ),
    );
  }
}
