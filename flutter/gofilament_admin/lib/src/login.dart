import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'i18n.dart';

import 'api.dart';
import 'controller.dart';
import 'theme.dart';

/// Brand mark: the brand logo when there is one, otherwise its initials.
class BrandMark extends StatelessWidget {
  const BrandMark({super.key, required this.admin, this.size = 40});

  final AdminController admin;
  final double size;

  @override
  Widget build(BuildContext context) {
    final palette = AdminPalette.of(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final brand = admin.brand;
    final logo =
        '${(dark ? brand['logo_dark_url'] : null) ?? brand['icon_url'] ?? brand['logo_url'] ?? ''}';

    if (logo.isNotEmpty) {
      return ClipRRect(
        borderRadius: BorderRadius.circular(size * 0.25),
        child: Image.network(logo,
            width: size,
            height: size,
            fit: BoxFit.contain,
            errorBuilder: (_, __, ___) => _initials(palette)),
      );
    }
    return _initials(palette);
  }

  Widget _initials(AdminPalette palette) {
    final words = admin.brandName
        .trim()
        .split(RegExp(r'\s+'))
        .where((w) => w.isNotEmpty)
        .toList();
    final text = words.isEmpty
        ? 'A'
        : (words.length == 1
            ? words[0].substring(0, words[0].length >= 2 ? 2 : 1)
            : words[0][0] + words[1][0]);
    return Container(
      width: size,
      height: size,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(size * 0.28),
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [palette.color('primary', 600), palette.color('accent', 600)],
        ),
      ),
      child: Text(text.toUpperCase(),
          style: TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.w700,
              fontSize: size * 0.38)),
    );
  }
}

/// Sign-in screen, branded from the public boot payload.
/// Demo account shown as a one-click "fill in" button on the sign-in screen.
///
/// It is opt-in: pass it to [GoFilamentAdmin.demoLogin] only from a demo or
/// showcase app. The shared package never contains credentials, and by
/// default the button only appears in debug builds, so a demo account can
/// never ship to production by accident.
class DemoLogin {
  const DemoLogin({
    required this.email,
    required this.password,
    this.label = 'Use demo account',
    this.debugOnly = true,
  });

  final String email;
  final String password;

  /// Button text.
  final String label;

  /// When true (the default), the button is only shown in debug builds.
  /// Set it to false only for a public demo deployment whose account is
  /// meant to be shared.
  final bool debugOnly;

  /// Whether the button should be shown in this build.
  bool get isAvailable => !debugOnly || kDebugMode;
}

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key, required this.admin, this.demoLogin});

  final AdminController admin;

  /// Optional demo account button (see [DemoLogin]).
  final DemoLogin? demoLogin;

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _form = GlobalKey<FormState>();
  final _username = TextEditingController();
  final _password = TextEditingController();
  bool _remember = false;
  bool _obscure = true;
  final _code = TextEditingController();
  bool _busy = false;
  String? _error;

  /// Set after the password step when a two-factor code is needed.
  String? _challenge;

  @override
  void dispose() {
    _username.dispose();
    _password.dispose();
    _code.dispose();
    super.dispose();
  }

  Future<void> _submitCode() async {
    final code = _code.text.trim();
    if (_busy || code.isEmpty) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.admin.loginTwoFactor(_challenge!, code);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _error = e.message;
          // An expired challenge starts over from the password.
          if (e.message.contains('expired')) _challenge = null;
        });
      }
    } catch (_) {
      if (mounted) {
        setState(
            () => _error = tr('Could not reach the server. Please try again.'));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _forgotPassword() async {
    final message = await showDialog<String>(
      context: context,
      builder: (_) => _ForgotPasswordDialog(
          admin: widget.admin, email: _username.text.trim()),
    );
    if (message != null && mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(message)));
    }
  }

  Future<void> _submit() async {
    if (_busy || !(_form.currentState?.validate() ?? false)) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final challenge = await widget.admin
          .login(_username.text.trim(), _password.text, remember: _remember);
      if (challenge != null && mounted) {
        setState(() {
          _challenge = challenge;
          _code.clear();
        });
      }
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _error = e.isUnauthorized
            ? tr('The username or password is incorrect.')
            : e.message);
      }
    } catch (_) {
      if (mounted) {
        setState(
            () => _error = tr('Could not reach the server. Please try again.'));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final admin = widget.admin;
    final theme = Theme.of(context);
    final palette = AdminPalette.of(context);
    final brand = admin.brand;
    final tagline = '${brand['tagline'] ?? ''}';

    final card = Card(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Form(
          key: _form,
          child: AutofillGroup(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(children: [
                  BrandMark(admin: admin, size: 44),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(admin.brandName,
                              style: theme.textTheme.titleLarge
                                  ?.copyWith(fontWeight: FontWeight.w700)),
                          if (tagline.isNotEmpty)
                            Text(tagline,
                                style: theme.textTheme.bodySmall?.copyWith(
                                    color: theme.colorScheme.onSurfaceVariant)),
                        ]),
                  ),
                ]),
                const SizedBox(height: 24),
                Text(tr('Sign in'),
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w600)),
                const SizedBox(height: 4),
                Text(tr('Enter your details to continue.'),
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
                const SizedBox(height: 20),
                if (_error != null) ...[
                  Container(
                    key: const Key('login-error'),
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color:
                          palette.color('danger', 500).withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Text(_error!,
                        style: TextStyle(color: theme.colorScheme.error)),
                  ),
                  const SizedBox(height: 16),
                ],
                if (_challenge != null) ...[
                  Text(
                      admin.accountFeatures['recovery_codes'] == true
                          ? tr(
                              'Enter the 6-digit code from your authenticator app, or one of your recovery codes.')
                          : tr(
                              'Enter the 6-digit code from your authenticator app.'),
                      style: theme.textTheme.bodyMedium),
                  const SizedBox(height: 14),
                  TextFormField(
                    key: const Key('login-code'),
                    controller: _code,
                    autofocus: true,
                    autofillHints: const [AutofillHints.oneTimeCode],
                    onFieldSubmitted: (_) => _submitCode(),
                    decoration: InputDecoration(
                        labelText: tr('Authentication code'),
                        prefixIcon: Icon(Icons.pin_outlined)),
                  ),
                  const SizedBox(height: 16),
                  FilledButton(
                    key: const Key('login-code-submit'),
                    onPressed: _busy ? null : _submitCode,
                    child: _busy
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2))
                        : Text(tr('Verify')),
                  ),
                  TextButton(
                    key: const Key('login-code-back'),
                    onPressed: _busy
                        ? null
                        : () => setState(() {
                              _challenge = null;
                              _error = null;
                            }),
                    child: Text(tr('Back to sign in')),
                  ),
                ] else ...[
                  TextFormField(
                    key: const Key('login-username'),
                    controller: _username,
                    autofillHints: const [AutofillHints.username],
                    textInputAction: TextInputAction.next,
                    decoration: InputDecoration(
                        labelText: tr('Username or email'),
                        prefixIcon: Icon(Icons.person_outline)),
                    validator: (v) => (v == null || v.trim().isEmpty)
                        ? tr('Enter your username or email')
                        : null,
                  ),
                  const SizedBox(height: 14),
                  TextFormField(
                    key: const Key('login-password'),
                    controller: _password,
                    obscureText: _obscure,
                    autofillHints: const [AutofillHints.password],
                    onFieldSubmitted: (_) => _submit(),
                    decoration: InputDecoration(
                      labelText: tr('Password'),
                      prefixIcon: const Icon(Icons.lock_outline),
                      suffixIcon: IconButton(
                        tooltip: _obscure
                            ? tr('Show password')
                            : tr('Hide password'),
                        icon: Icon(_obscure
                            ? Icons.visibility_outlined
                            : Icons.visibility_off_outlined),
                        onPressed: () => setState(() => _obscure = !_obscure),
                      ),
                    ),
                    validator: (v) => (v == null || v.isEmpty)
                        ? tr('Enter your password')
                        : null,
                  ),
                  const SizedBox(height: 6),
                  CheckboxListTile(
                    value: _remember,
                    onChanged: (v) => setState(() => _remember = v ?? false),
                    title: Text(tr('Keep me signed in')),
                    dense: true,
                    contentPadding: EdgeInsets.zero,
                    controlAffinity: ListTileControlAffinity.leading,
                  ),
                  const SizedBox(height: 10),
                  FilledButton(
                    key: const Key('login-submit'),
                    onPressed: _busy ? null : _submit,
                    child: _busy
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2))
                        : Text(tr('Sign in')),
                  ),
                  if (widget.demoLogin?.isAvailable ?? false) ...[
                    const SizedBox(height: 12),
                    OutlinedButton.icon(
                      key: const Key('demo-login-btn'),
                      icon: const Icon(Icons.bolt, size: 18),
                      label: Text(tr(widget.demoLogin!.label)),
                      onPressed: _busy
                          ? null
                          : () => setState(() {
                                _username.text = widget.demoLogin!.email;
                                _password.text = widget.demoLogin!.password;
                                _error = null;
                              }),
                    ),
                  ],
                  if (admin.accountFeatures['registration'] == true)
                    TextButton(
                      key: const Key('login-register'),
                      onPressed: _busy
                          ? null
                          : () => Navigator.of(context).push(
                              MaterialPageRoute<void>(
                                  builder: (_) =>
                                      RegisterScreen(admin: widget.admin))),
                      child: Text(tr('No account yet? Create one')),
                    ),
                  if (admin.accountFeatures['password_reset'] == true)
                    TextButton(
                      key: const Key('login-forgot'),
                      onPressed: _busy ? null : _forgotPassword,
                      child: Text(tr('Forgot your password?')),
                    ),
                ],
              ],
            ),
          ),
        ),
      ),
    );

    return Scaffold(
      body: Container(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [
              palette.color(
                  'primary', theme.brightness == Brightness.dark ? 950 : 50),
              theme.scaffoldBackgroundColor,
              palette.color(
                  'accent', theme.brightness == Brightness.dark ? 950 : 50),
            ],
          ),
        ),
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(20),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(mainAxisSize: MainAxisSize.min, children: [
                card,
                if (brand['hide_powered_by'] != true) ...[
                  const SizedBox(height: 16),
                  Text(tr('Powered by GoFilament'),
                      style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant)),
                ],
              ]),
            ),
          ),
        ),
      ),
    );
  }
}

class _ForgotPasswordDialog extends StatefulWidget {
  const _ForgotPasswordDialog({required this.admin, required this.email});
  final AdminController admin;
  final String email;

  @override
  State<_ForgotPasswordDialog> createState() => _ForgotPasswordDialogState();
}

class _ForgotPasswordDialogState extends State<_ForgotPasswordDialog> {
  late final _email = TextEditingController(
      text: widget.email.contains('@') ? widget.email : '');
  String? _error;
  bool _busy = false;

  @override
  void dispose() {
    _email.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    setState(() => _busy = true);
    try {
      final msg = await widget.admin.api.forgotPassword(_email.text.trim());
      if (mounted) Navigator.pop(context, msg);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } catch (_) {
      if (mounted) setState(() => _error = tr('Could not reach the server.'));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: Text(tr('Reset your password')),
        content: SizedBox(
          width: 360,
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            Text(tr(
                'Enter your email address and we will send you a link to choose a new password.')),
            const SizedBox(height: 12),
            TextField(
              key: const Key('forgot-email'),
              controller: _email,
              keyboardType: TextInputType.emailAddress,
              decoration: InputDecoration(
                  labelText: tr('Email address'), errorText: _error),
              onSubmitted: (_) => _send(),
            ),
          ]),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: Text(tr('Cancel'))),
          FilledButton(
            key: const Key('forgot-send'),
            onPressed: _busy ? null : _send,
            child: Text(tr('Send link')),
          ),
        ],
      );
}

/// Self-registration (Panel.Registration with Accounts.Register).
class RegisterScreen extends StatefulWidget {
  const RegisterScreen({super.key, required this.admin});
  final AdminController admin;

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final _form = GlobalKey<FormState>();
  final _name = TextEditingController();
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _confirm = TextEditingController();
  bool _busy = false;
  String? _error;
  Map<String, String> _fieldErrors = const {};

  @override
  void dispose() {
    for (final c in [_name, _email, _password, _confirm]) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _submit() async {
    if (_busy || !(_form.currentState?.validate() ?? false)) return;
    setState(() {
      _busy = true;
      _error = null;
      _fieldErrors = const {};
    });
    try {
      await widget.admin.register(
          _name.text.trim(), _email.text.trim(), _password.text, _confirm.text);
      if (mounted) Navigator.of(context).pop();
    } on ApiException catch (e) {
      if (!mounted) return;
      final errs = e.body?['errors'];
      setState(() {
        _error = e.message;
        _fieldErrors =
            errs is Map ? errs.map((k, v) => MapEntry('$k', '$v')) : const {};
      });
      if (e.status == 200 && mounted) Navigator.of(context).pop();
    } catch (_) {
      if (mounted) {
        setState(
            () => _error = tr('Could not reach the server. Please try again.'));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    InputDecoration deco(String label, String field) =>
        InputDecoration(labelText: label, errorText: _fieldErrors[field]);
    return Scaffold(
      appBar: AppBar(title: Text(tr('Create an account'))),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(20),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 420),
            child: Card(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Form(
                  key: _form,
                  child: AutofillGroup(
                    child: Column(
                        mainAxisSize: MainAxisSize.min,
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          if (_error != null && _fieldErrors.isEmpty) ...[
                            Text(_error!,
                                key: const Key('register-error'),
                                style:
                                    TextStyle(color: theme.colorScheme.error)),
                            const SizedBox(height: 12),
                          ],
                          TextFormField(
                            key: const Key('register-name'),
                            controller: _name,
                            autofillHints: const [AutofillHints.name],
                            decoration: deco('Name', 'name'),
                            validator: (v) => (v ?? '').trim().isEmpty
                                ? tr('Enter your name')
                                : null,
                          ),
                          const SizedBox(height: 12),
                          TextFormField(
                            key: const Key('register-email'),
                            controller: _email,
                            keyboardType: TextInputType.emailAddress,
                            autofillHints: const [AutofillHints.email],
                            decoration: deco(tr('Email address'), 'email'),
                            validator: (v) => (v ?? '').contains('@')
                                ? null
                                : tr('Enter a valid email address'),
                          ),
                          const SizedBox(height: 12),
                          TextFormField(
                            key: const Key('register-password'),
                            controller: _password,
                            obscureText: true,
                            autofillHints: const [AutofillHints.newPassword],
                            decoration: deco('Password', 'password'),
                            validator: (v) => (v ?? '').length < 8
                                ? tr('Use at least 8 characters')
                                : null,
                          ),
                          const SizedBox(height: 12),
                          TextFormField(
                            key: const Key('register-confirm'),
                            controller: _confirm,
                            obscureText: true,
                            decoration: deco(tr('Confirm password'), ''),
                            validator: (v) => v != _password.text
                                ? tr('The passwords do not match')
                                : null,
                            onFieldSubmitted: (_) => _submit(),
                          ),
                          const SizedBox(height: 20),
                          FilledButton(
                            key: const Key('register-submit'),
                            onPressed: _busy ? null : _submit,
                            child: Text(tr('Create account')),
                          ),
                        ]),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
