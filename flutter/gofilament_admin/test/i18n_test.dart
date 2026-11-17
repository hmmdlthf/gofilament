import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'fake_panel.dart';

void main() {
  tearDown(() => AdminStrings.use(const {}));

  testWidgets('the app shows its own words in the panel language',
      (tester) async {
    final api = FakePanel().api()..locale = 'de';
    await tester.pumpWidget(GoFilamentAdmin(api: api));
    await tester.pumpAndSettle();
    expect(find.text('Anmelden'), findsWidgets);
    expect(find.text('Benutzername oder E-Mail'), findsOneWidget);
  });

  test('tr fills placeholders and falls back to English', () {
    AdminStrings.use({'{n} selected': '{n} ausgewählt'});
    expect(tr('{n} selected', {'n': 3}), '3 ausgewählt');
    expect(tr('Sign out'), 'Sign out');
  });
}
