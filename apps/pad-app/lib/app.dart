import 'package:flutter/material.dart';
import 'package:pad_app/ui_kit/kid_theme.dart';
import 'package:pad_app/task_board/page.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/word_playback/controller.dart';

class StudyClawPadApp extends StatelessWidget {
  const StudyClawPadApp({
    super.key,
    this.autoLoad = true,
    this.initialDate,
    this.initialApiBaseUrl,
    this.initialFamilyId,
    this.initialUserId,
    this.repository = const RemoteTaskBoardRepository(),
    this.wordPlaybackController,
  });

  final bool autoLoad;
  final String? initialDate;
  final String? initialApiBaseUrl;
  final int? initialFamilyId;
  final int? initialUserId;
  final TaskBoardRepository repository;
  final WordPlaybackController? wordPlaybackController;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'StudyClaw Pad',
      debugShowCheckedModeBanner: false,
      theme: KidTheme.light,
      home: PadTaskBoardPage(
        autoLoad: autoLoad,
        initialDate: initialDate,
        initialApiBaseUrl: initialApiBaseUrl,
        initialFamilyId: initialFamilyId,
        initialUserId: initialUserId,
        repository: repository,
        wordPlaybackController: wordPlaybackController,
      ),
    );
  }
}
