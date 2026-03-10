class DailyStats {
  const DailyStats({
    required this.period,
    required this.startDate,
    required this.endDate,
    required this.totals,
    required this.encouragement,
  });

  factory DailyStats.fromJson(Map<String, dynamic> json) {
    return DailyStats(
      period: json['period']?.toString() ?? '',
      startDate: json['start_date']?.toString() ?? '',
      endDate: json['end_date']?.toString() ?? '',
      totals: StatsTotals.fromJson(json['totals'] as Map<String, dynamic>? ?? const {}),
      encouragement: json['encouragement']?.toString() ?? '',
    );
  }

  final String period;
  final String startDate;
  final String endDate;
  final StatsTotals totals;
  final String encouragement;
}

class StatsTotals {
  const StatsTotals({
    required this.totalTasks,
    required this.completedTasks,
    required this.pendingTasks,
    required this.completionRate,
    required this.autoPoints,
    required this.manualPoints,
    required this.totalPointsDelta,
    required this.pointsBalance,
    required this.wordItems,
    required this.completedWordItems,
    required this.dictationSessions,
  });

  factory StatsTotals.fromJson(Map<String, dynamic> json) {
    return StatsTotals(
      totalTasks: _readInt(json['total_tasks']),
      completedTasks: _readInt(json['completed_tasks']),
      pendingTasks: _readInt(json['pending_tasks']),
      completionRate: (json['completion_rate'] as num?)?.toDouble() ?? 0.0,
      autoPoints: _readInt(json['auto_points']),
      manualPoints: _readInt(json['manual_points']),
      totalPointsDelta: _readInt(json['total_points_delta']),
      pointsBalance: _readInt(json['points_balance']),
      wordItems: _readInt(json['word_items']),
      completedWordItems: _readInt(json['completed_word_items']),
      dictationSessions: _readInt(json['dictation_sessions']),
    );
  }

  final int totalTasks;
  final int completedTasks;
  final int pendingTasks;
  final double completionRate;
  final int autoPoints;
  final int manualPoints;
  final int totalPointsDelta;
  final int pointsBalance;
  final int wordItems;
  final int completedWordItems;
  final int dictationSessions;

  int get completionRatePercent => (completionRate * 100).round();

  String get pointsDeltaLabel => totalPointsDelta >= 0 ? '+$totalPointsDelta' : '$totalPointsDelta';
}

int _readInt(Object? value) {
  if (value is int) {
    return value;
  }
  if (value is num) {
    return value.toInt();
  }
  return int.tryParse(value?.toString() ?? '') ?? 0;
}
