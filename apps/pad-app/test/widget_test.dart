import 'package:flutter_test/flutter_test.dart';
import 'package:pad_app/main.dart';

void main() {
  testWidgets('renders pad task board shell', (tester) async {
    await tester.pumpWidget(const StudyClawPadApp(autoLoad: false));

    expect(find.text('孩子任务同步台'), findsOneWidget);
    expect(find.text('同步配置'), findsOneWidget);
    expect(find.text('加载任务板'), findsOneWidget);
  });
}
