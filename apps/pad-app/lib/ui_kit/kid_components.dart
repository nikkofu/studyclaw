import 'package:flutter/material.dart';
import 'package:pad_app/ui_kit/kid_theme.dart';

class KidSmallBtn extends StatelessWidget {
  const KidSmallBtn({super.key, required this.label, required this.color, this.onTap});
  final String label; final Color color; final VoidCallback? onTap;
  @override Widget build(BuildContext context) => GestureDetector(onTap: onTap, child: Container(padding: const EdgeInsets.symmetric(vertical: 16), decoration: BoxDecoration(color: onTap == null ? Colors.grey.shade300 : color, borderRadius: BorderRadius.circular(16), border: Border.all(color: KidColors.black, width: 2)), child: Center(child: Text(label, style: const TextStyle(color: KidColors.white, fontWeight: FontWeight.w900, fontSize: 16)))));
}

class KidActionBtn extends StatelessWidget {
  const KidActionBtn({super.key, required this.label, required this.color, this.onTap});
  final String label; final Color color; final VoidCallback? onTap;
  @override Widget build(BuildContext context) => GestureDetector(onTap: onTap, child: Container(padding: const EdgeInsets.symmetric(vertical: 16), decoration: BoxDecoration(color: onTap == null ? Colors.grey.shade300 : color, borderRadius: BorderRadius.circular(16), border: Border.all(color: KidColors.black, width: 2)), child: Center(child: Text(label, style: const TextStyle(color: KidColors.white, fontWeight: FontWeight.w900, fontSize: 18)))));
}

class KidMetricTile extends StatelessWidget {
  const KidMetricTile({super.key, required this.label, required this.value, required this.icon, required this.color});
  final String label, value; final IconData icon; final Color color;
  @override Widget build(BuildContext context) => Container(padding: const EdgeInsets.all(24), decoration: BoxDecoration(color: color, borderRadius: BorderRadius.circular(24), border: Border.all(color: KidColors.black, width: 3)), child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [Container(padding: const EdgeInsets.all(8), decoration: const BoxDecoration(color: KidColors.white, shape: BoxShape.circle), child: Icon(icon, color: color, size: 24)), const SizedBox(height: 16), Text(value, style: const TextStyle(fontSize: 32, fontWeight: FontWeight.w900, color: KidColors.white)), Text(label, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w800, color: KidColors.white))]));
}

class KidInlineLoading extends StatelessWidget {
  const KidInlineLoading({super.key, required this.title, required this.description});
  final String title, description;
  @override Widget build(BuildContext context) => Center(child: Column(mainAxisSize: MainAxisSize.min, children: [const CircularProgressIndicator(color: KidColors.color1), const SizedBox(height: 16), Text(title, style: const TextStyle(fontWeight: FontWeight.w800, fontSize: 18)), Text(description)]));
}

class KidInlineError extends StatelessWidget {
  const KidInlineError({super.key, required this.title, required this.description});
  final String title, description;
  @override Widget build(BuildContext context) => Center(child: Column(mainAxisSize: MainAxisSize.min, children: [const Icon(Icons.error_outline_rounded, color: KidColors.color5, size: 48), const SizedBox(height: 16), Text(title, style: const TextStyle(fontWeight: FontWeight.w800, fontSize: 18)), Text(description)]));
}

class KidBottomSheetFrame extends StatelessWidget {
  const KidBottomSheetFrame({super.key, required this.title, required this.child});
  final String title; final Widget child;
  @override Widget build(BuildContext context) => Container(padding: const EdgeInsets.fromLTRB(24, 8, 24, 40), decoration: const BoxDecoration(color: Colors.white, borderRadius: BorderRadius.vertical(top: Radius.circular(32))), child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.start, children: [Center(child: Container(width: 40, height: 4, decoration: BoxDecoration(color: KidColors.black.withAlpha(40), borderRadius: BorderRadius.circular(2)))), const SizedBox(height: 24), Text(title, style: const TextStyle(fontSize: 24, fontWeight: FontWeight.w900)), const SizedBox(height: 24), child]));
}
