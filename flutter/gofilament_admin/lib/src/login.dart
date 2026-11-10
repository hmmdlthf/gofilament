import 'package:flutter/material.dart';

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
class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key, required this.admin});

  final AdminController admin;

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _form = GlobalKey<FormState>();
  final _username = TextEditingController();
  final _password = TextEditingController();
  bool _remember = false;
  bool _obscure = true;
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _username.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_busy || !(_form.currentState?.validate() ?? false)) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.admin
          .login(_username.text.trim(), _password.text, remember: _remember);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _error = e.isUnauthorized
            ? 'The username or password is incorrect.'
            : e.message);
      }
    } catch (_) {
      if (mounted) {
        setState(
            () => _error = 'Could not reach the server. Please try again.');
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
                Text('Sign in',
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w600)),
                const SizedBox(height: 4),
                Text('Enter your details to continue.',
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
                TextFormField(
                  key: const Key('login-username'),
                  controller: _username,
                  autofillHints: const [AutofillHints.username],
                  textInputAction: TextInputAction.next,
                  decoration: const InputDecoration(
                      labelText: 'Username or email',
                      prefixIcon: Icon(Icons.person_outline)),
                  validator: (v) => (v == null || v.trim().isEmpty)
                      ? 'Enter your username or email'
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
                    labelText: 'Password',
                    prefixIcon: const Icon(Icons.lock_outline),
                    suffixIcon: IconButton(
                      tooltip: _obscure ? 'Show password' : 'Hide password',
                      icon: Icon(_obscure
                          ? Icons.visibility_outlined
                          : Icons.visibility_off_outlined),
                      onPressed: () => setState(() => _obscure = !_obscure),
                    ),
                  ),
                  validator: (v) =>
                      (v == null || v.isEmpty) ? 'Enter your password' : null,
                ),
                const SizedBox(height: 6),
                CheckboxListTile(
                  value: _remember,
                  onChanged: (v) => setState(() => _remember = v ?? false),
                  title: const Text('Keep me signed in'),
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
                      : const Text('Sign in'),
                ),
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
                  Text('Powered by GoFilament',
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
