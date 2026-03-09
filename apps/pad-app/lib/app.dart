import 'package:flutter/material.dart';
import 'package:pad_app/task_board/page.dart';
import 'package:pad_app/task_board/repository.dart';

class StudyClawPadApp extends StatelessWidget {
  const StudyClawPadApp({
    super.key,
    this.autoLoad = true,
    this.initialDate,
    this.repository = const RemoteTaskBoardRepository(),
  });

  final bool autoLoad;
  final String? initialDate;
  final TaskBoardRepository repository;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'StudyClaw Pad',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF0F766E)),
        scaffoldBackgroundColor: const Color(0xFFF3F7F5),
      ),
      home: PadTaskBoardPage(
        autoLoad: autoLoad,
        initialDate: initialDate,
        repository: repository,
      ),
    );
  }
}
