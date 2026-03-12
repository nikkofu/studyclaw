import 'package:flutter/material.dart';
import 'package:pad_app/ui_kit/kid_theme.dart';

class KidCard extends StatelessWidget {
  const KidCard({
    super.key,
    required this.child,
    this.color = KidColors.white,
    this.borderRadius = 24.0,
    this.borderColor = KidColors.black,
    this.padding = const EdgeInsets.all(24.0),
    this.hasBorder = true,
  });

  final Widget child;
  final Color color;
  final double borderRadius;
  final Color borderColor;
  final EdgeInsets padding;
  final bool hasBorder;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: padding,
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(borderRadius),
        border: hasBorder 
          ? Border.all(color: borderColor, width: 3.0) // Strong bold flat border
          : null,
      ),
      child: child,
    );
  }
}
