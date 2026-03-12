import 'package:pad_app/task_board/daily_stats.dart';
import 'package:pad_app/task_board/models.dart';

enum TaskCompletionKind { singleTask, homeworkGroup, subjectGroup, allTasks }

String? buildTaskCompletionEncouragement({
  required TaskCompletionKind kind,
  required TaskBoard? previousBoard,
  required TaskBoard board,
  DailyStats? dailyStats,
  String? subject,
  String? groupTitle,
  String? taskContent,
}) {
  if (previousBoard == null || board.summary.total <= 0) {
    return null;
  }

  final completedDelta =
      board.summary.completed - previousBoard.summary.completed;
  if (completedDelta <= 0) {
    return null;
  }

  if (board.summary.completed >= board.summary.total) {
    final statsEncouragement = dailyStats?.encouragement.trim() ?? '';
    if (statsEncouragement.isNotEmpty) {
      return statsEncouragement;
    }
    return '今天的挑战全部完成啦！你认真坚持到了最后，真棒。';
  }

  final progressMessage = _buildProgressMessage(
    completed: board.summary.completed,
    total: board.summary.total,
    pending: board.summary.pending,
  );

  switch (kind) {
    case TaskCompletionKind.singleTask:
      final target = _bestTargetLabel(
        taskContent: taskContent,
        groupTitle: groupTitle,
        subject: subject,
      );
      if (_looksChallenging(target)) {
        return '这一步不轻松，你还是认真拿下了。$progressMessage';
      }
      if (target.isNotEmpty) {
        return '“$target”完成啦。$progressMessage';
      }
      return '又完成了一小步。$progressMessage';
    case TaskCompletionKind.homeworkGroup:
      final target = (groupTitle ?? '').trim();
      if (_looksChallenging(target)) {
        return '“$target”这一步不轻松，你还是稳稳完成了。$progressMessage';
      }
      if (target.isNotEmpty) {
        return '“$target”这一组完成啦。$progressMessage';
      }
      return '这一组任务完成啦。$progressMessage';
    case TaskCompletionKind.subjectGroup:
      final target = (subject ?? '').trim();
      if (target.isNotEmpty) {
        return '$target 这一科推进了一大步。$progressMessage';
      }
      return '这一科任务完成啦。$progressMessage';
    case TaskCompletionKind.allTasks:
      return dailyStats?.encouragement.trim().isNotEmpty == true
          ? dailyStats!.encouragement.trim()
          : '今天的挑战全部完成啦！你认真坚持到了最后，真棒。';
  }
}

String _buildProgressMessage({
  required int completed,
  required int total,
  required int pending,
}) {
  final progressLabel = '现在已经完成 $completed/$total 项';
  if (completed <= 1) {
    return '$progressLabel，开了个好头，继续保持。';
  }
  if (pending <= 1) {
    return '$progressLabel，离全部完成只差最后一步。';
  }
  if (completed * 2 >= total) {
    return '$progressLabel，已经过半啦，继续稳稳向前。';
  }
  return '$progressLabel，继续一点点往前推进。';
}

String _bestTargetLabel({
  String? taskContent,
  String? groupTitle,
  String? subject,
}) {
  final taskValue = (taskContent ?? '').trim();
  if (taskValue.isNotEmpty) {
    return taskValue;
  }

  final groupValue = (groupTitle ?? '').trim();
  if (groupValue.isNotEmpty) {
    return groupValue;
  }

  return (subject ?? '').trim();
}

bool _looksChallenging(String raw) {
  final value = raw.trim();
  if (value.isEmpty) {
    return false;
  }

  const keywords = <String>[
    '订正',
    '复习',
    '默写',
    '听写',
    '错题',
    '背诵',
    '练习',
    '口算',
    '作文',
  ];

  for (final keyword in keywords) {
    if (value.contains(keyword)) {
      return true;
    }
  }
  return false;
}
