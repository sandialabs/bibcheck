let pdfDoc = null;
let pageNum = 1;
let pageRendering = false;
let pageNumPending = null;
const scale = 1.5;

// PDF.js setup
pdfjsLib.GlobalWorkerOptions.workerSrc = 'https://cdnjs.cloudflare.com/ajax/libs/pdf.js/3.11.174/pdf.worker.min.js';

// Load PDF
pdfjsLib.getDocument(`/uploads/${filename}`).promise.then(function(pdf) {
    pdfDoc = pdf;
    document.getElementById('page-info').textContent = `Page ${pageNum} of ${pdf.numPages}`;
    renderPage(pageNum);
});

function renderPage(num) {
    pageRendering = true;
    pdfDoc.getPage(num).then(function(page) {
        const viewport = page.getViewport({scale: scale});
        const canvas = document.getElementById('pdf-canvas');
        const context = canvas.getContext('2d');
        canvas.height = viewport.height;
        canvas.width = viewport.width;

        const renderContext = {
            canvasContext: context,
            viewport: viewport
        };

        const renderTask = page.render(renderContext);
        renderTask.promise.then(function() {
            pageRendering = false;
            if (pageNumPending !== null) {
                renderPage(pageNumPending);
                pageNumPending = null;
            }
        });
    });
}

function queueRenderPage(num) {
    if (pageRendering) {
        pageNumPending = num;
    } else {
        renderPage(num);
    }
}

document.getElementById('prev-page').addEventListener('click', function() {
    if (pageNum <= 1) return;
    pageNum--;
    queueRenderPage(pageNum);
    document.getElementById('page-info').textContent = `Page ${pageNum} of ${pdfDoc.numPages}`;
});

document.getElementById('next-page').addEventListener('click', function() {
    if (pageNum >= pdfDoc.numPages) return;
    pageNum++;
    queueRenderPage(pageNum);
    document.getElementById('page-info').textContent = `Page ${pageNum} of ${pdfDoc.numPages}`;
});

// Load and update bibliography entries
function loadEntries() {
    fetch(`/api/entries/${docID}`)
        .then(response => response.json())
        .then(entries => {
            const grid = document.getElementById('entries-grid');
            grid.innerHTML = '';

            entries.forEach(entry => {
                const row = document.createElement('div');
                row.className = 'entry-row';

                const textCell = document.createElement('div');
                textCell.className = `entry-cell ${entry.text_status}`;
                textCell.textContent = entry.text;
                if (entry.text_status === 'pending' || entry.text_status === 'active') {
                    textCell.innerHTML = '<div class="spinner"></div>';
                } else {
                    textCell.textContent = entry.text;
                }

                const analysisCell = document.createElement('div');
                analysisCell.className = `entry-cell ${entry.analysis_status} ${entry.analysis_found}`;
                if (entry.analysis_status === 'pending' || entry.analysis_status === 'active') {
                    analysisCell.innerHTML = '<div class="spinner"></div>';
                } else {
                    analysisCell.textContent = entry.analysis;
                }

                row.appendChild(textCell);
                row.appendChild(analysisCell);
                grid.appendChild(row);
            });
        });
}

// Poll for updates
let pollId;
pollId = setInterval(function() {
    fetch(`/api/status/${docID}`)
        .then(response => response.json())
        .then(data => {
            loadEntries();
            console.log(`${data.completed} / ${data.total}`)
            if (data.completed === data.total && data.total > 0) {
                console.log("clearInterval(this)")
                clearInterval(pollId);
            }
        });
}, 2000);

// Initial load
loadEntries();
