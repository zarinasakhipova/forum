// Этот файл JavaScript пока пуст, но его можно использовать для добавления
// интерактивности на стороне клиента
document.addEventListener('DOMContentLoaded', function() {
    // Найти все секции с комментариями
    document.querySelectorAll('.space-y-3').forEach(function(commentSection) {
        commentSection.scrollTop = commentSection.scrollHeight;
    });
});